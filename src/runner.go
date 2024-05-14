package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

type Runner struct {
	clickHouseClient ClickHouseClient
	verifierCreds    VerifierConfig
	httpClient       http.Client
	goroutineTimeout time.Duration
	producerWorkers  int
	consumerWorkers  int
}

func NewRunner(config RunnerConfig) *Runner {
	clickHouseClient, err := NewClickHouseClient(config.ClickHouseConfig)
	if err != nil {
		return nil
	}
	var verifierCreds = config.VerifierCreds
	return &Runner{
		clickHouseClient: *clickHouseClient,
		verifierCreds:    verifierCreds,
		httpClient: http.Client{
			Timeout: time.Duration(config.VerifierTimeout) * time.Second,
		},
		goroutineTimeout: time.Duration(config.GoroutineTimeout),
		producerWorkers:  config.ProducerWorkers,
		consumerWorkers:  config.ConsumerWorkers,
	}
}

func (runner Runner) SendGetRequest(url string) (*VerificationResponse, int) {
	response, err := runner.httpClient.Get(url)
	if err != nil {
		fmt.Printf("Gotten error %s", err)
		return &VerificationResponse{}, 599
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Gotten error %s", err)
		return &VerificationResponse{}, 599
	}
	var result VerificationResponse
	json.Unmarshal(body, &result)
	if result.Score == nil {
		result.Score = &NAN
	}
	return &result, response.StatusCode
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
			link, _ := task.CreateVerifyGetRequestLink()
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
			// TODO(sokunkov): Add len batch on runner variables
			if len(batch) == 500 {
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
	results := make(chan VerificationResult, runner.consumerWorkers)
	for i := 0; i < runner.producerWorkers; i++ {
		wg.Add(1)
		go runner.producer(&tasks, &results, &wg)
	}
	var numTasks int = len(verifyParamsList)
	bar := progressbar.Default(int64(numTasks))
	for i := 0; i < runner.consumerWorkers; i++ {
		wg.Add(1)
		go runner.consumer(&results, &numTasks, &wg, bar)
	}
	defer close(results)
	defer close(tasks)
	wg.Wait()
	fmt.Println("Finished!")
}
