package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Runner[S StoredValueType, R ResponseType[S, P], P ParamsType] struct {
	clickHouseClient     ClickhouseClient[S, P]
	verifierCreds        VerifierConfig
	httpClient           *resty.Client
	workerTimeout        time.Duration
	httpRetries          Retries
	runConfig            RunConfig
	selectRetries        Retries
	logger               *zap.SugaredLogger
	qualityControlConfig QualityControlConfig
}

func NewRunner[S StoredValueType, R ResponseType[S, P], P ParamsType](
	config RunnerConfig,
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

	logger.Infow("Creating table which is required for the run")
	// TODO(evgenymng): uncomment, when actual DDL is written
	// var zeroInstance S
	// zeroInstance.GetCreateQuery()

	httpClient := initializeHttpClient(config, logger)

	runner := Runner[S, R, P]{
		clickHouseClient: *clickHouseClient,
		verifierCreds:    config.VerifierConfig,
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

func (r *Runner[S, R, P]) SendGetRequest(
	ctx context.Context,
	req GetRequest[P],
) ([]S, error) {
	url, err := req.CreateGetRequestLink(r.runConfig.ExtraParams)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, "unsuccessResponses", nil)
	lastResponse, err := r.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
	}

	unsuccessResponses := lastResponse.
		Request.
		Context().
		Value("unsuccessResponses").([]*resty.Response)
	responses := unsuccessResponses
	if lastResponse.IsSuccess() || r.httpRetries.NumRetries == 0 ||
		lastResponse.StatusCode() == 0 {
		responses = append(responses, lastResponse)
	}

	resultList := []S{}
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
			json.Unmarshal(response.Body(), &result)
		}
		storedValue := result.IntoWith(req.Params, i+1, url, statusCode)

		resultList = append(
			resultList,
			storedValue,
		)
	}
	return resultList, nil
}

// Run the runner's job within a given context.
func (r *Runner[S, R, P]) Run(ctx context.Context) {
	start := time.Now()

	results := make(chan S, r.runConfig.SelectionBatchSize)
	tasks := make(chan GetRequest[P], r.runConfig.SelectionBatchSize)
	defer close(results)
	defer close(tasks)

	numRequests := int64(0)
	numSuccesses := int64(0)
	numSuccessesWithScore := int64(0)
	workerCtx, cancel := context.WithTimeout(ctx, r.workerTimeout)
	defer cancel()
	batchSignals := make(chan bool)
	close(batchSignals)

	for i := 0; i < r.runConfig.ProducerWorkers; i++ {
		go r.producer(
			workerCtx,
			i,
			tasks,
			results,
			&numRequests,
			&numSuccesses,
		)
	}
	for i := 0; i < r.runConfig.ConsumerWorkers; i++ {
		go r.consumer(workerCtx, i, results, batchSignals)
	}

	// batch processing start time
	bpStartTime := time.Now()
	paramsBucket := []P{}

	for {
		select {
		case _, ok := <-batchSignals:
			if !ok {
				return
			}
			standby := false

			// check the batch processing time
			sinceBatchStart := time.Since(bpStartTime)
			if sinceBatchStart > time.Duration(
				r.qualityControlConfig.Period,
			)*time.Second {
				r.logger.Infow(
					"Batch processing takes longer than it should. "+
						"The runner is entering standby mode.",
					"num_successes", numSuccesses,
					"num_requests", numRequests,
					"num_successes_with_scores", numSuccessesWithScore,
					"tag", TagQualityControl,
				)
				standby = true
			}

			// check the number of successful responses
			if numSuccesses < int64(
				float64(numRequests)*r.qualityControlConfig.Threshold,
			) {
				r.logger.Infow(
					"Too many 5xx errors from the verifier. "+
						"The runner is entering standby mode.",
					"tag", TagQualityControl,
				)
				standby = true
			}

			// check if we have something left to do
			if len(paramsBucket) == 0 {
				var paramsList []P
				err := retry.Do(
					func() error {
						var err error
						paramsList, err = r.clickHouseClient.SelectNextBatch(
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
						"Failed to fetch verification parameters from the ClickHouse!",
						"error",
						err,
						"tag",
						TagClickHouseError,
					)
					break
				}

				if len(paramsList) == 0 {
					r.logger.Infow(
						"Processing of the task stream for the given time "+
							"period is completed! "+
							"The runner is entering standby mode.",
						"tag",
						TagRunnerDebug,
					)
					standby = true
				} else {
					r.logger.Infow(
						"Successfully retrieved batch from the ClickHouse!",
						"batch_size",
						len(paramsList),
						"tag",
						TagClickHouseSuccess,
					)
					paramsBucket = paramsList
				}
			}

			numRequests = 0
			numSuccesses = 0
			numSuccessesWithScore = 0
			bpStartTime = time.Now()
			if standby {
				err := r.standby(ctx)
				if err != nil {
					return
				}
			}

			// start processing the next batch
			for _, verifyParams := range paramsBucket[:r.runConfig.VerificationBatchSize] {
				verifyGetRequest := NewGetRequest(
					r.verifierCreds.Host,
					r.verifierCreds.Port,
					r.verifierCreds.Method,
					verifyParams,
				)
				tasks <- *verifyGetRequest
			}
			paramsBucket = paramsBucket[r.runConfig.VerificationBatchSize:]

		case <-workerCtx.Done():
			r.logger.Warnw(
				"Workers timeout has been reached. "+
					"The runner is entering standby mode.",
				"elapsed", r.workerTimeout,
				"tag", TagRunnerStandby,
			)
			err := r.standby(ctx)
			if err != nil {
				return
			}

		case <-ctx.Done():
			r.logger.Debug(
				"Parent context is cancelled, shutting down the runner job.",
				"tag", TagRunnerDebug,
			)
			r.logger.Debugw(
				"Run finished!",
				"tag", TagRunnerDebug,
				"run_time", time.Since(start),
			)
			return
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

func initializeHttpClient(
	config RunnerConfig,
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
		AddRetryHook(
			func(r *resty.Response, err error) {
				ctx := r.Request.Context()
				responses := ctx.Value("unsuccessResponses").([]*resty.Response)
				responses = append(responses, r)
				newCtx := context.WithValue(
					ctx,
					"unsuccessResponses",
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
	tasks chan GetRequest[P],
	results chan S,
	numRequests *int64,
	numSuccesses *int64,
	// numSuccessesWithScores *int64,
) {
	for {
		select {
		case task, ok := <-tasks:
			if !ok {
				// the channel is closed
				return
			}
			resultList, err := r.SendGetRequest(ctx, task)
			if err != nil {
				r.logger.Debugw(
					"There was an error, while sending the request "+
						"to the subject API",
					"error", err,
				)
				// something is wrong with that task
				break
			}

			for _, result := range resultList {
				results <- result
				atomic.AddInt64(numRequests, 1)
				// TODO(nrydanov): Return it back in some way
				if result.GetStatusCode() == 200 {
					atomic.AddInt64(numSuccesses, 1)
					// if *result.Response.Score != NAN {
					// 	atomic.AddInt64(numSuccessesWithScores, 1)
					// }
				}
			}
		case <-ctx.Done():
			r.logger.Debugw(
				"Producer's context is cancelled",
				"producer_num", producerNum,
				"error", ctx.Err(),
			)
			return
		}
	}
}

func (r *Runner[S, R, P]) consumer(
	ctx context.Context,
	consumerNum int,
	results chan S,
	batchSignals chan bool,
) {
	var batch []S
	for {
		select {
		case result, ok := <-results:
			if !ok {
				// the channel is closed
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
				batch = []S{}
				batchSignals <- true
			}
		case <-ctx.Done():
			r.logger.Debugw(
				"Consumer's context is cancelled",
				"consumer_num", consumerNum,
				"error", ctx.Err(),
			)
			return
		}
	}
}
