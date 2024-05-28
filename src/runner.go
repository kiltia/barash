package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Runner struct {
	clickHouseClient     ClickHouseClient
	verifierCreds        VerifierConfig
	httpClient           *resty.Client
	httpRetries          Retries
	goroutineTimeout     time.Duration
	runConfig            RunConfig
	selectRetries        Retries
	logger               *zap.SugaredLogger
	qualityControlConfig QualityControlConfig
}

func NewRunner(config RunnerConfig) *Runner {
	logger := zap.Must(config.LoggerConfig.Build()).Sugar()

	clickHouseClient, version, err := NewClickHouseClient(config.ClickHouseConfig)
	if err != nil {
		logger.Errorw(
			"Connection to the ClickHouse database was unsuccessful!",
			"error", err,
			"tag", CLICKHOUSE_ERROR_TAG,
		)
		return nil
	} else {
		logger.Infow(
			"Connection to the ClickHouse database was successful!",
			"tag", CLICKHOUSE_SUCCESS_TAG,
		)
		logger.Infow(
			fmt.Sprintf("%v", version),
			"tag", CLICKHOUSE_SUCCESS_TAG,
		)
	}

	httpClient := resty.New().SetRetryCount(config.HttpRetries.NumRetries).
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
						"tag", ERROR_RESPONSE_TAG,
					)
					return true
				}
				return false
			},
		).AddRetryHook(
		func(r *resty.Response, err error) {
			ctx := r.Request.Context()
			responses := ctx.Value("unsuccessResponses").([]*resty.Response)
			responses = append(responses, r)
			newCtx := context.WithValue(ctx, "unsuccessResponses", responses)
			r.Request.SetContext(newCtx)
		},
	).SetLogger(logger)
	runner := Runner{
		clickHouseClient:     *clickHouseClient,
		verifierCreds:        config.VerifierConfig,
		httpClient:           httpClient,
		goroutineTimeout:     time.Duration(config.Timeouts.GoroutineTimeout),
		runConfig:            config.RunConfig,
		httpRetries:          config.HttpRetries,
		selectRetries:        config.SelectRetries,
		qualityControlConfig: config.QualityControlConfig,
		logger:               logger,
	}
	return &runner
}

func (runner Runner) SendGetRequest(verifyGetRequest VerifyGetRequest) []VerificationResult {
	url, _ := verifyGetRequest.CreateVerifyGetRequestLink(runner.runConfig.ExtraParams)
	ctx := context.Background()
	ctx = context.WithValue(ctx, "unsuccessResponses", make([]*resty.Response, 0))
	lastResponse, _ := runner.httpClient.R().SetContext(ctx).Get(url)
	unsuccessResponses := lastResponse.Request.Context().Value("unsuccessResponses").([]*resty.Response)
	responses := unsuccessResponses
	if lastResponse.IsSuccess() || runner.httpRetries.NumRetries == 0 || lastResponse.StatusCode() == 0 {
		responses = append(responses, lastResponse)
	}
	verificationResultList := make([]VerificationResult, 0)
	for i, response := range responses {
		var result VerificationResponse
		statusCode := response.StatusCode()
		if statusCode == 0 {
			result = VerificationResponse{Score: &NAN}
			statusCode = 599
			runner.logger.Debugw(
				"Timeout was reached while waiting for a request",
				"url", url,
				"error", "TIMEOUT REACHED",
				"tag", RESPONSE_TIMEOUT_TAG,
			)
		} else {
			json.Unmarshal(response.Body(), &result)
			if result.Score == nil {
				result.Score = &NAN
			}
		}
		verificationResultList = append(
			verificationResultList,
			VerificationResult{
				AttemptsNumber:       i + 1,
				VerifyParams:         verifyGetRequest.VerifyParams,
				VerificationResponse: &result,
				VerificationLink:     url,
				StatusCode:           statusCode,
			},
		)
	}
	return verificationResultList
}

func (runner Runner) producer(
	producerNum int,
	tasks *chan VerifyGetRequest,
	results *chan VerificationResult,
	wg *sync.WaitGroup,
	numRequests *int64,
	numSuccesses *int64,
) {
	for loop := true; loop; {
		select {
		case task, ok := <-*tasks:
			if !ok {
				break
			}
			verificationResultList := runner.SendGetRequest(task)
			for _, verificationResult := range verificationResultList {
				*results <- verificationResult
				atomic.AddInt64(numRequests, 1)
				if verificationResult.StatusCode == 200 {
					atomic.AddInt64(numSuccesses, 1)
				}
			}
		case <-time.After(runner.goroutineTimeout * time.Second):
			loop = false
		}
	}
	runner.logger.Debugw(
		"Producer finished his work!",
		"producer_num", producerNum,
		"tag", RUNNER_DEBUG_TAG,
	)
	wg.Done()
}

func (runner Runner) consumer(consumerNum int, results *chan VerificationResult, wg *sync.WaitGroup) {
	var batch []VerificationResult
	for loop := true; loop; {
		select {
		case result, ok := <-*results:
			if !ok {
				break
			}
			batch = append(batch, result)
			if len(batch) >= runner.runConfig.InsertionBatchSize {
				err := runner.clickHouseClient.AsyncInsertBatch(batch, runner.runConfig.Tag)
				if err != nil {
					runner.logger.Errorw(
						"Insertion to the ClickHouse database was unsuccessful!",
						"error", err,
						"consumer_num", consumerNum,
						"tag", CLICKHOUSE_ERROR_TAG,
					)
					continue
				}
				runner.logger.Infow(
					"Insertion to the ClickHouse database was successful!",
					"batch_len", len(batch),
					"consumer_num", consumerNum,
					"tag", CLICKHOUSE_SUCCESS_TAG,
				)
				batch = make([]VerificationResult, 0)
			}
		case <-time.After(runner.goroutineTimeout * time.Second):
			loop = false
		}
	}
	if len(batch) != 0 {
		runner.clickHouseClient.AsyncInsertBatch(batch, runner.runConfig.Tag)
	}
	runner.logger.Debugw(
		"Consumer finished his work!",
		"consumer_num", consumerNum,
		"tag", RUNNER_DEBUG_TAG,
	)
	wg.Done()
}

func (runner Runner) Run() {
	start := time.Now()
	var wg sync.WaitGroup
	results := make(chan VerificationResult, runner.runConfig.SelectionBatchSize)
	tasks := make(chan VerifyGetRequest, runner.runConfig.SelectionBatchSize)
	var numRequests int64 = 0
	var numSuccesses int64 = 0
	for i := 0; i < runner.runConfig.ProducerWorkers; i++ {
		wg.Add(1)
		go runner.producer(i, &tasks, &results, &wg, &numRequests, &numSuccesses)
	}
	for i := 0; i < runner.runConfig.ConsumerWorkers; i++ {
		wg.Add(1)
		go runner.consumer(i, &results, &wg)
	}
	var verifyParamsBucket []VerifyParams = make([]VerifyParams, 0)
	timeOnAwait := 0
	batchesProcessingStartTime := time.Now()
	for {
		if timeOnAwait >= int(runner.goroutineTimeout) {
			runner.logger.Warn(
				"While waiting, the goroutines completed their work.",
				"time", timeOnAwait,
				"tag", RUNNER_STANDBY_TAG,
			)
			break
		}
		if len(tasks) == 0 {
			timeSinceBatchesProcessingStart := time.Since(batchesProcessingStartTime)
			if timeSinceBatchesProcessingStart > time.Duration(runner.qualityControlConfig.Period)*time.Second {
				if numSuccesses < int64(float64(numRequests)*runner.qualityControlConfig.Threshold) {
					runner.logger.Infow(
						"Too many 5xx errors from the verifier. The main thread goes into standby mode.",
						"time", runner.runConfig.SleepTime,
						"tag", RUNNER_STANDBY_TAG,
					)
					timeOnAwait += runner.runConfig.SleepTime
					time.Sleep(time.Duration(runner.runConfig.SleepTime) * time.Second)
					runner.logger.Infow(
						"The main thread has exited standby mode.",
						"time", runner.runConfig.SleepTime,
						"tag", RUNNER_STANDBY_TAG,
					)
				}
				numRequests = 0
				numSuccesses = 0
				batchesProcessingStartTime = time.Now()
			}
			if len(verifyParamsBucket) == 0 {
				var verifyParamsList *[]VerifyParams
				err := retry.Do(
					func() error {
						selectedList, err := runner.clickHouseClient.SelectNextBatch(
							runner.runConfig.DayOffset,
							runner.runConfig.SelectionBatchSize,
						)
						verifyParamsList = selectedList
						return err
					},
					retry.Attempts(uint(runner.selectRetries.NumRetries)+1),
				)
				if err != nil {
					runner.logger.Errorw(
						"Select verification params from clickhouse was unsuccess!",
						"error", err,
						"tag", CLICKHOUSE_ERROR_TAG,
					)
					break
				}
				if len(*verifyParamsList) == 0 {
					runner.logger.Infow(
						"Processing of the stream for the specified time is completed! Main thread enter on standby mode.",
						"time", runner.runConfig.SleepTime,
						"tag", RUNNER_STANDBY_TAG,
					)
					time.Sleep(time.Duration(runner.runConfig.SleepTime) * time.Second)
					runner.logger.Infow(
						"The main thread has exited standby mode.",
						"time", runner.runConfig.SleepTime,
						"tag", RUNNER_STANDBY_TAG,
					)
					timeOnAwait += runner.runConfig.SleepTime
					continue
				}
				runner.logger.Infow(
					"Batch from the ClickHouse database was received successfully!",
					"batch_len", len(*verifyParamsList),
					"tag", CLICKHOUSE_SUCCESS_TAG,
				)
				verifyParamsBucket = *verifyParamsList
			}
			timeOnAwait = 0
			for _, verifyParams := range verifyParamsBucket[:runner.runConfig.VerificationBatchSize] {
				verifyGetRequest := NewVerifyGetRequest(
					runner.verifierCreds.Host,
					runner.verifierCreds.Port,
					runner.verifierCreds.Method,
					verifyParams,
				)

				tasks <- *verifyGetRequest
			}
			verifyParamsBucket = verifyParamsBucket[runner.runConfig.VerificationBatchSize:]
		}
	}
	defer close(results)
	defer close(tasks)
	wg.Wait()
	runner.logger.Debugw(
		"Run finished!",
		"tag", RUNNER_DEBUG_TAG,
		"run_time", time.Since(start),
	)
}
