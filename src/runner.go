package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"orb/runner/src/config"
	"orb/runner/src/util"

	"github.com/avast/retry-go/v4"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Runner[S StoredValueType, R ResponseType[S, P], P ParamsType] struct {
	clickHouseClient     ClickhouseClient[S, P]
	apiConfig            config.ApiConfig
	httpClient           *resty.Client
	workerTimeout        time.Duration
	httpRetries          config.RetryConfig
	runConfig            config.RunConfig
	selectRetries        config.RetryConfig
	logger               *zap.SugaredLogger
	qualityControlConfig config.QualityControlConfig
}

func NewRunner[S StoredValueType, R ResponseType[S, P], P ParamsType](
	config config.RunnerConfig,
) *Runner[S, R, P] {
	logger := zap.Must(config.LoggerConfig.Build()).Sugar()

	clickHouseClient, version, err := NewClickHouseClient[S, P](
		config.ClickHouseConfig,
	)
	if err != nil {
		logger.Errorw(
			"Connection to the ClickHouse database was unsuccessful!",
			"error", err,
			"tag", TagClickHouseError,
		)
		return nil
	} else {
		logger.Infow(
			"Connection to the ClickHouse database was successful!",
			"tag", TagClickHouseSuccess,
		)
		logger.Infow(
			fmt.Sprintf("%v", version),
			"tag", TagClickHouseSuccess,
		)
	}

	// logger.Infow("Creating table which is required for the run")
	// TODO(evgenymng): uncomment, when actual DDL is written
	// var zeroInstance S
	// zeroInstance.GetCreateQuery()

	httpClient := initHttpClient(config, logger)

	runner := Runner[S, R, P]{
		clickHouseClient: *clickHouseClient,
		apiConfig:        config.ApiConfig,
		httpClient:       httpClient,
		workerTimeout: time.Duration(
			config.Timeouts.GoroutineTimeout,
		) * time.Second,
		runConfig:            config.RunConfig,
		httpRetries:          config.HttpRetries,
		selectRetries:        config.SelectRetries,
		qualityControlConfig: config.QualityControlConfig,
		logger:               logger,
	}
	return &runner
}

type RequestContextKey int

const (
	RequestContextKeyUnsuccessfulResponses RequestContextKey = iota
)

func (r *Runner[S, R, P]) SendGetRequest(
	ctx context.Context,
	req GetRequest[P],
) ([]S, error) {
	url, err := req.CreateGetRequestLink(r.runConfig.ExtraParams)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(
		ctx,
		RequestContextKeyUnsuccessfulResponses,
		[]*resty.Response{},
	)
	lastResponse, err := r.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
	}

	responses := lastResponse.
		Request.
		Context().
		Value(RequestContextKeyUnsuccessfulResponses).([]*resty.Response)
	if lastResponse.IsSuccess() || r.httpRetries.NumRetries == 0 ||
		lastResponse.StatusCode() == 0 {
		responses = append(responses, lastResponse)
	}

	results := []S{}
	for i, response := range responses {
		var result R
		statusCode := response.StatusCode()
		if statusCode == 0 {
			result = *new(R)
			statusCode = 599
			r.logger.Debugw(
				"Timeout was reached while waiting for a request",
				"url", url,
				"error", "TIMEOUT REACHED",
				"tag", TagResponseTimeout,
			)
		} else {
			err = json.Unmarshal(response.Body(), &result)
			if err != nil {
				return nil, err
			}
		}
		storedValue := result.IntoWith(req.Params, i+1, url, statusCode)

		results = append(
			results,
			storedValue,
		)
	}
	return results, nil
}

func (r *Runner[S, R, P]) controlQuality(
	bpStartTime time.Time,
	processedBatch *[]S,
) bool {
	numRequests := len(*processedBatch)
	numSuccesses := util.Reduce(
		util.Map(*processedBatch, func(report S) int {
			return report.GetStatusCode()
		}),
		0,
		func(acc int, v int) int {
			if v == 200 {
				return acc + 1
			}
			return acc
		},
	)

	sinceBatchStart := time.Since(bpStartTime)
	standby := false
	// NOTE(nrydanov): Case 1. Batch processing takes too much time
	if sinceBatchStart > time.Duration(
		r.qualityControlConfig.Period,
	)*time.Second {
		r.logger.Infow(
			"Batch processing takes longer than it should. "+
				"The runner is entering standby mode.",
			"num_successes", numSuccesses,
			"num_requests", numRequests,
			"tag", TagQualityControl,
		)
		standby = true
	}

	// NOTE(nrydanov): Case 2. Too many requests ends with errors
	if numSuccesses < int(
		float64(numRequests)*r.qualityControlConfig.Threshold,
	) {
		r.logger.Infow(
			"Too many 5xx errors from the API. "+
				"The runner is entering standby mode.",
			"tag", TagQualityControl,
		)
		standby = true
	}

	return standby
}

// Run the runner's job within a given context.
func (r *Runner[S, R, P]) Run(ctx context.Context) {
	produced := make(chan S, r.runConfig.SelectionBatchSize)
	consumed := make(chan []S, 1)
	tasks := make(chan *GetRequest[P], r.runConfig.SelectionBatchSize)
	nothingLeft := make(chan bool, 1)
	defer close(produced)
	defer close(tasks)
	defer close(consumed)
	defer close(nothingLeft)

	workerCtx, cancel := context.WithCancel(ctx)
    defer cancel()

	for i := 0; i < r.runConfig.ProducerWorkers; i++ {
		go r.producer(
			workerCtx,
			i,
			tasks,
			produced,
			nothingLeft,
		)
	}
	for i := 0; i < r.runConfig.ConsumerWorkers; i++ {
		go r.consumer(workerCtx, i, produced)
	}

	selectedBatch := []P{}
	bpStartTime := time.Now()

	nothingLeft <- true

	for {
		select {
		case _, ok := <-nothingLeft:

			r.logger.Debug("Got \"nothing left\" signal from one of producers")
			if !ok {
				// TODO(nrydanov): What should we do when channels are closed?
				return
			}

			err := retry.Do(
				func() error {
					var err error
					selectedBatch, err = r.clickHouseClient.SelectNextBatch(
						ctx,
						r.runConfig.DayOffset,
						r.runConfig.SelectionBatchSize,
					)
					return err
				},
				retry.Attempts(uint(r.selectRetries.NumRetries)+1),
			)
			if err != nil {
				r.logger.Errorw(
					"Failed to fetch URL parameters from the ClickHouse!",
					"error",
					err,
					"tag",
					TagClickHouseError,
				)
				break
			}
			r.logger.Debug("Creating tasks for a producers")
			for _, task := range selectedBatch[:r.runConfig.VerificationBatchSize] {
				tasks <- NewGetRequest(r.apiConfig.Host,
					r.apiConfig.Port,
					r.apiConfig.Method,
					task)
			}
			tasks <- nil
			selectedBatch = selectedBatch[r.runConfig.VerificationBatchSize:]

		case consumed, ok := <-consumed:
			if !ok {
				// TODO(nrydanov): What should we do when channels are closed?
				return
			}

			standby := r.controlQuality(bpStartTime, &consumed)
			// TODO(nrydanov): Check if standby still works when runner has
			// nothing to do
			if standby {
				err := r.standby(ctx)
				if err != nil {
					return
				}
			}

			bpStartTime = time.Now()
		}
	}
}

func (r *Runner[S, R, P]) standby(ctx context.Context) error {
	r.logger.Infow("The runner has entered standby mode.")
	waitTime := time.Duration(r.runConfig.SleepTime) * time.Second
	defer r.logger.Infow("The runner has left standby mode")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		return nil
	}
}

func initHttpClient(
	config config.RunnerConfig,
	logger *zap.SugaredLogger,
) *resty.Client {
	return resty.New().SetRetryCount(config.HttpRetries.NumRetries).
		SetTimeout(time.Duration(time.Duration(config.Timeouts.VerifierTimeout) * time.Second)).
		SetRetryWaitTime(time.Duration(config.HttpRetries.MinWaitTime) * time.Second).
		SetRetryMaxWaitTime(time.Duration(config.HttpRetries.MaxWaitTime) * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				if r.StatusCode() >= http.StatusInternalServerError {
					logger.Debugw(
						"Retrying request",
						"request_status_code", r.StatusCode(),
						"verify_url", r.Request.URL,
						"tag", TagErrorResponse,
					)
					return true
				}
				return false
			},
		).
		// TODO(nrydanov): Find other way to handle list of unsucessful responses
		// as using WithValue for these purposes is anti-pattern
		AddRetryHook(
			func(r *resty.Response, err error) {
				ctx := r.Request.Context()
				responses := ctx.Value(RequestContextKeyUnsuccessfulResponses).([]*resty.Response)
				responses = append(responses, r)
				newCtx := context.WithValue(
					ctx,
					RequestContextKeyUnsuccessfulResponses,
					responses,
				)
				r.Request.SetContext(newCtx)
			},
		).
		SetLogger(logger)
}

func (r *Runner[S, R, P]) producer(
	ctx context.Context,
	producerNum int,
	tasks chan *GetRequest[P],
	results chan S,
	nothingLeft chan bool,
) {
	for {
		select {
		case task, ok := <-tasks:
			if !ok {
				// TODO(nrydanov): What should we do when channels are closed?
				return
			}
			if task == nil {
				r.logger.Infow(
					"Producer has no work left, asking for a new batch",
					"producer_num",
					producerNum,
				)
				nothingLeft <- true
				break
			}

			r.logger.Debugw(
				"Sending request to get page contents",
				"producer_num",
				producerNum,
			)
			resultList, err := r.SendGetRequest(ctx, *task)
			if err != nil {
				r.logger.Debugw(
					"There was an error, while sending the request "+
						"to the subject API",
					"error", err,
				)
				break
			}

			for _, result := range resultList {
				results <- result
			}
		}
	}
}

func (r *Runner[S, R, P]) consumer(
	ctx context.Context,
	consumerNum int,
	results chan S,
) {
	var batch []S
	for {
		select {
		case result, ok := <-results:
			if !ok {
				return
			}
			batch = append(batch, result)
			if len(batch) >= r.runConfig.InsertionBatchSize {
				err := r.clickHouseClient.AsyncInsertBatch(
					ctx,
					batch,
					r.runConfig.Tag,
				)
				if err != nil {
					r.logger.Errorw(
						"Insertion to the ClickHouse database was unsuccessful!",
						"error",
						err,
						"consumer_num",
						consumerNum,
						"tag",
						TagClickHouseError,
					)
					break
				}
				r.logger.Infow(
					"Insertion to the ClickHouse database was successful!",
					"batch_len", len(batch),
					"consumer_num", consumerNum,
					"tag", TagClickHouseSuccess,
				)
				batch = make([]S, 0)

			}
		}
	}
}
