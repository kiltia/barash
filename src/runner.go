package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/schollz/progressbar/v3"
)

type Runner struct {
	clickHouseClient ClickHouseClient
	verifierCreds    VerifierConfig
	httpClient       *resty.Client
	goroutineTimeout time.Duration
	runConfig        RunConfig
}

func NewRunner(config RunnerConfig) *Runner {
	clickHouseClient, err := NewClickHouseClient(config.ClickHouseConfig)
	if err != nil {
		return nil
	}
	var verifierCreds = config.VerifierConfig
	httpClient := resty.NewWithClient(&http.Client{
		Timeout: time.Duration(config.Timeouts.VerifierTimeout) * time.Second,
	}).SetRetryCount(config.Retries.NumRetries).
		SetRetryWaitTime(time.Duration(config.Retries.MinWaitTime) * time.Second).
		SetRetryMaxWaitTime(time.Duration(config.Retries.MaxWaitTime) * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				return r.StatusCode() >= http.StatusInternalServerError
			},
		)
	var runner = Runner{
		clickHouseClient: *clickHouseClient,
		verifierCreds:    verifierCreds,
		httpClient:       httpClient,
		goroutineTimeout: time.Duration(config.Timeouts.GoroutineTimeout),
		runConfig:        config.RunConfig,
	}
	return &runner
}

func (runner Runner) SendGetRequest(url string) (*VerificationResponse, int) {
	response, err := runner.httpClient.R().Get(url)
	if response.StatusCode() == 0 {
		fmt.Printf("Gotten error %s", err)
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
	tasks *chan VerifyGetRequest,
	results *chan VerificationResult,
	wg *sync.WaitGroup,
) {
	for len(*tasks) != 0 {
		select {
		case task, ok := <-*tasks:
			if !ok {
				break
			}
			link, _ := task.CreateVerifyGetRequestLink(runner.runConfig.ExtraParams)
			response, statusCode := runner.SendGetRequest(link)
			*results <- VerificationResult{
				VerifyParams:         task.VerifyParams,
				VerificationResponse: response,
				VerificationLink:     link,
				StatusCode:           statusCode,
			}
		case <-time.After(runner.goroutineTimeout * time.Second):
			break
		}
	}
	wg.Done()
}

func (runner Runner) consumer(
	results *chan VerificationResult,
	numTasks *int,
	wg *sync.WaitGroup,
	bar *progressbar.ProgressBar,
) {
	var batch []VerificationResult
	for *numTasks != 0 {
		select {
		case result, ok := <-*results:
			if !ok {
				break
			}
			batch = append(batch, result)
			if len(batch) == runner.runConfig.BatchSize {
				runner.clickHouseClient.AsyncInsertBatch(batch)
				batch = make([]VerificationResult, 0)
			}
			bar.Add(1)
			*numTasks = *numTasks - 1
		case <-time.After(runner.goroutineTimeout * time.Second):
			break
		}
	}
	if len(batch) != 0 {
		runner.clickHouseClient.AsyncInsertBatch(batch)
	}
	wg.Done()
}

func (runner Runner) Run(verifyParamsList []VerifyParams) {
	tasks := make(chan VerifyGetRequest, len(verifyParamsList))
	var wg sync.WaitGroup
	for _, verifyParams := range verifyParamsList {
		verifyGetRequest := NewVerifyGetRequest(
			runner.verifierCreds.Host,
			runner.verifierCreds.Port,
			runner.verifierCreds.Method,
			verifyParams,
		)

		tasks <- *verifyGetRequest
	}
	results := make(chan VerificationResult, runner.runConfig.ConsumerWorkers)
	for i := 0; i < runner.runConfig.ProducerWorkers; i++ {
		wg.Add(1)
		go runner.producer(&tasks, &results, &wg)
	}
	var numTasks int = len(verifyParamsList)
	bar := progressbar.Default(int64(numTasks))
	for i := 0; i < runner.runConfig.ConsumerWorkers; i++ {
		wg.Add(1)
		go runner.consumer(&results, &numTasks, &wg, bar)
	}
	defer close(results)
	defer close(tasks)
	wg.Wait()
	fmt.Println("Finished!")
}
