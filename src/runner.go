package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Runner struct {
	clickHouseClient ClickHouseClient
	verifierCreds    VerifierConfig
	httpClient       *resty.Client
	goroutineTimeout time.Duration
	runConfig        RunConfig
	logger           *zap.SugaredLogger
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

	httpClient := resty.NewWithClient(
		&http.Client{
			Timeout: time.Duration(config.Timeouts.VerifierTimeout) * time.Second,
		},
	).SetRetryCount(config.Retries.NumRetries).
		SetRetryWaitTime(time.Duration(config.Retries.MinWaitTime) * time.Second).
		SetRetryMaxWaitTime(time.Duration(config.Retries.MaxWaitTime) * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				return r.StatusCode() >= http.StatusInternalServerError
			},
		)

	runner := Runner{
		clickHouseClient: *clickHouseClient,
		verifierCreds:    config.VerifierConfig,
		httpClient:       httpClient,
		goroutineTimeout: time.Duration(config.Timeouts.GoroutineTimeout),
		runConfig:        config.RunConfig,
		logger:           logger,
	}
	return &runner
}

func (runner Runner) logReport(producerNum int, result VerificationResult) {
	switch {
	case result.StatusCode == 599:
		runner.logger.Debugw(
			"Timeout was reached while waiting for a request",
			"url", result.VerificationLink,
			"error", "TIMEOUT REACHED",
			"producer_num", producerNum,
			"tag", RESPONSE_TIMEOUT_TAG,
		)
		break
	case result.StatusCode != 200:
		runner.logger.Debugw(
			"Error response gotten from backend",
			"url", result.VerificationLink,
			"error", *result.VerificationResponse.Error,
			"producer_num", producerNum,
			"tag", ERROR_RESPONSE_TAG,
		)
		break
	case result.StatusCode == 200 && result.VerificationResponse.Score == &NAN:
		runner.logger.Debugw(
			"Fail response gotten from backend",
			"url", result.VerificationLink,
			"fail", *result.VerificationResponse.DebugInfo.CrawlerDebug.FailStatus,
			"producer_num", producerNum,
			"tag", FAIL_RESPONSE_TAG,
		)
		break
	default:
		runner.logger.Debugw(
			"Request was success",
			"url", result.VerificationLink,
			"producer_num", producerNum,
			"tag", SUCCESS_RESPONSE_TAG,
		)
		break
	}
}

func (runner Runner) SendGetRequest(url string) (*VerificationResponse, int) {
	response, _ := runner.httpClient.R().Get(url)
	if response.StatusCode() == 0 {
		return &VerificationResponse{Score: &NAN}, 599
	}
	var result VerificationResponse
	json.Unmarshal(response.Body(), &result)
	if result.Score == nil {
		result.Score = &NAN
	}
	return &result, response.StatusCode()
}

func (runner Runner) producer(
	producerNum int,
	tasks *chan VerifyGetRequest,
	results *chan VerificationResult,
	wg *sync.WaitGroup,
) {
	for loop := true; loop; {
		select {
		case task, ok := <-*tasks:
			if !ok {
				break
			}
			link, _ := task.CreateVerifyGetRequestLink(runner.runConfig.ExtraParams)
			response, statusCode := runner.SendGetRequest(link)
			result := VerificationResult{
				VerifyParams:         task.VerifyParams,
				VerificationResponse: response,
				VerificationLink:     link,
				StatusCode:           statusCode,
			}
			runner.logReport(producerNum, result)
			*results <- result
		case <-time.After(runner.goroutineTimeout * time.Second):
			loop = false
			break
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
			// TODO(sokunkov): Сome up with a condition to stop the worker
			if len(batch) >= 500 {
				err := runner.clickHouseClient.AsyncInsertBatch(batch)
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
			break
		}
	}
	if len(batch) != 0 {
		runner.clickHouseClient.AsyncInsertBatch(batch)
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
	results := make(chan VerificationResult, runner.runConfig.ConsumerWorkers)
	selectionBatchSize := runner.runConfig.BatchSize * runner.runConfig.ConsumerWorkers
	tasks := make(chan VerifyGetRequest, 10)
	for i := 0; i < runner.runConfig.ProducerWorkers; i++ {
		wg.Add(1)
		go runner.producer(i, &tasks, &results, &wg)
	}
	for i := 0; i < runner.runConfig.ConsumerWorkers; i++ {
		wg.Add(1)
		go runner.consumer(i, &results, &wg)
	}
	for {
		if len(tasks) == 0 {
			// TODO(sokunkov): Hard code
			verifyParamsList, err := runner.clickHouseClient.SelectNextBatch(30, selectionBatchSize)
			if err != nil {
				runner.logger.Errorw(
					"Select verification params from clickhouse was unsuccess!",
					"error", err,
					"tag", CLICKHOUSE_ERROR_TAG,
				)
				break
			}
			if len(*verifyParamsList) == 0 {
				break
			}
			runner.logger.Infow(
				"Batch from the ClickHouse database was received successfully!",
				"batch_len", len(*verifyParamsList),
				"tag", CLICKHOUSE_SUCCESS_TAG,
			)
			for _, verifyParams := range *verifyParamsList {
				verifyGetRequest := NewVerifyGetRequest(
					runner.verifierCreds.Host,
					runner.verifierCreds.Port,
					runner.verifierCreds.Method,
					verifyParams,
				)

				tasks <- *verifyGetRequest
			}
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
