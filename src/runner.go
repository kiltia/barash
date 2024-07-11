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

type FetcherResult[S StoredValueType] struct {
	Value               S
	ProcessingStartTime time.Time
}

func NewFetcherResult[S StoredValueType](
	value S,
	processingStartTime time.Time,
) FetcherResult[S] {
	return FetcherResult[S]{
		Value:               value,
		ProcessingStartTime: processingStartTime,
	}
}

type ProcessedBatch[S StoredValueType] struct {
	Batch               []S
	ProcessingStartTime time.Time
}

func NewProcessedBatch[S StoredValueType](
	batch []S,
	processingStartTime time.Time,
) ProcessedBatch[S] {
	return ProcessedBatch[S]{
		Batch:               batch,
		ProcessingStartTime: processingStartTime,
	}
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

// Run the runner's job within a given context.
func (r *Runner[S, R, P]) Run(ctx context.Context) {
	// + 1 for the [nil] task
	fetcherTasks := make(chan *GetRequest[P], r.runConfig.SelectionBatchSize+1)
	fetcherResults := make(
		chan FetcherResult[S],
		r.runConfig.SelectionBatchSize,
	)
	writtenBatches := make(
		chan ProcessedBatch[S],
		r.runConfig.WriterWorkers,
	)

	nothingLeft := make(chan bool, 1)
	qcResults := make(chan int, 1)
	defer close(fetcherResults)
	defer close(fetcherTasks)
	defer close(writtenBatches)
	defer close(nothingLeft)

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < r.runConfig.FetcherWorkers; i++ {
		go r.fetcher(
			workerCtx,
			i,
			fetcherTasks,
			fetcherResults,
			nothingLeft,
		)
	}
	for i := 0; i < r.runConfig.WriterWorkers; i++ {
		go r.writer(workerCtx, i, fetcherResults, writtenBatches)
	}
	go r.qualityControl(workerCtx, writtenBatches, qcResults)

	selectedBatch := []P{}
	nothingLeft <- true

	for {
		select {
		case _, ok := <-nothingLeft:
			r.logger.Debug("Got \"nothing left\" signal from one of fetchers")
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
			r.logger.Debug("Creating tasks for a fetchers")
			for _, task := range selectedBatch[:r.runConfig.VerificationBatchSize] {
				fetcherTasks <- NewGetRequest(r.apiConfig.Host,
					r.apiConfig.Port,
					r.apiConfig.Method,
					task)
			}
			fetcherTasks <- nil
			selectedBatch = selectedBatch[r.runConfig.VerificationBatchSize:]

		case failCount, ok := <-qcResults:
			if !ok {
				return
			}

			// TODO(nrydanov): Check if standby still works when runner has
			// nothing to do
			// NOTE(evgenymng): I mean, if it has nothing to do, it will wait
			// for a signal from somewhere, effectively sleeping.
			if failCount > 0 {
				err := r.standby(ctx)
				if err != nil {
					return
				}
			} else {
				r.logger.Infow(
					"Batch quality control has successfully been passed",
					"tag", TagQualityControl,
				)
			}
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

func (r *Runner[S, R, P]) fetcher(
	ctx context.Context,
	fetcherNum int,
	tasks chan *GetRequest[P],
	results chan FetcherResult[S],
	nothingLeft chan bool,
) {
	for {
		select {
		case task, ok := <-tasks:
			startTime := time.Now()
			if !ok {
				// TODO(nrydanov): What should we do when channels are closed?
				return
			}
			if task == nil {
				r.logger.Infow(
					"Fetcher has no work left, asking for a new batch",
					"fetcher_num", fetcherNum,
					"tag", TagRunnerDebug,
				)
				nothingLeft <- true
				break
			}

			r.logger.Debugw(
				"Sending request to get page contents",
				"fetcher_num",
				fetcherNum,
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
				results <- NewFetcherResult(result, startTime)
			}

		case <-ctx.Done():
			r.logger.Debugw(
				"Fetcher's context is cancelled",
				"fetcher_num", fetcherNum,
				"error", ctx.Err(),
			)
			return
		}
	}
}

func (r *Runner[S, R, P]) writer(
	ctx context.Context,
	consumerNum int,
	results chan FetcherResult[S],
	processedBatches chan ProcessedBatch[S],
) {
	var oldest *time.Time
	var batch []S
	for {
		select {
		case result, ok := <-results:
			if !ok {
				return
			}

			batch = append(batch, result.Value)
			if oldest == nil || result.ProcessingStartTime.Before(*oldest) {
				oldest = &result.ProcessingStartTime
			}

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

				processedBatches <- NewProcessedBatch(batch, *oldest)
				batch = []S{}
				oldest = nil
			}

		case <-ctx.Done():
			r.logger.Debugw(
				"Consumer's context is cancelled",
				"consumer_num", consumerNum,
				"error", ctx.Err(),
				"tag", TagRunnerDebug,
			)
			return
		}
	}
}

func (r *Runner[S, R, P]) qualityControl(
	ctx context.Context,
	processedBatches chan ProcessedBatch[S],
	qcResults chan int,
) {
	for {
		select {
		case batch, ok := <-processedBatches:
			if !ok {
				return
			}

			failCount := 0
			numRequests := len(batch.Batch)
			numSuccesses := util.Reduce(
				util.Map(batch.Batch, func(res S) int {
					return res.GetStatusCode()
				}),
				0,
				func(acc int, v int) int {
					if v == 200 {
						return acc + 1
					}
					return acc
				},
			)
			sinceBatchStart := time.Since(batch.ProcessingStartTime)
			// NOTE(nrydanov): Case 1. Batch processing takes too much time
			if sinceBatchStart > time.Duration(
				r.qualityControlConfig.Period,
			)*time.Second {
				r.logger.Infow(
					"Batch processing takes longer than it should. ",
					"num_successes", numSuccesses,
					"num_requests", numRequests,
					"tag", TagQualityControl,
				)
				failCount++
			}

			// NOTE(nrydanov): Case 2. Too many requests ends with errors
			if numSuccesses < int(
				float64(numRequests)*r.qualityControlConfig.Threshold,
			) {
				r.logger.Infow(
					"Too many 5xx errors from the API.",
					"tag", TagQualityControl,
				)
				failCount++
			}
			qcResults <- failCount

		case <-ctx.Done():
			r.logger.Debugw(
				"Quality control routine's context is cancelled",
				"error", ctx.Err(),
			)
			return
		}
	}
}
