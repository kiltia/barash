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
	}
}

func (runner Runner) SendGetRequest(url string) (*VerificationResult, int) {
	response, err := runner.httpClient.Get(url)
	if err != nil {
		fmt.Printf("Gotten error %s", err)
		return nil, 500
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Gotten error %s", err)
		return nil, 500
	}
	var result VerificationResult
	json.Unmarshal(body, &result)
	return &result, response.StatusCode
}

func (runner Runner) producer(tasks *chan VerifyGetRequest, results *chan Triple, wg *sync.WaitGroup) {
	for len(*tasks) != 0 {
		select {
		case task, ok := <-*tasks:
			if !ok {
				break
			}
			link, _ := task.CreateVerifyGetRequestLink()
			result, statusCode := runner.SendGetRequest(link)
			*results <- Triple{VerifyParams: task.VerifyParams, VerificationResult: *result, StatusCode: statusCode}
		case <-time.After(runner.goroutineTimeout * time.Second):
			break
		}
	}
	wg.Done()
}

func (runner Runner) consumer(results *chan Triple, numTasks *int, wg *sync.WaitGroup, bar *progressbar.ProgressBar) {
	var batch []Triple
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
				batch = make([]Triple, 0)
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

func (runner Runner) Run(producerSize int, consumerSize int, verifyGetRequestList []VerifyGetRequest) {
	tasks := make(chan VerifyGetRequest, len(verifyGetRequestList))
	var wg sync.WaitGroup
	for _, task := range verifyGetRequestList {
		tasks <- task
	}
	results := make(chan Triple, consumerSize)
	for i := 0; i < producerSize; i++ {
		wg.Add(1)
		go runner.producer(&tasks, &results, &wg)
	}
	var numTasks int = len(verifyGetRequestList)
	bar := progressbar.Default(int64(numTasks))
	for i := 0; i < consumerSize; i++ {
		wg.Add(1)
		go runner.consumer(&results, &numTasks, &wg, bar)
	}
	defer close(results)
	defer close(tasks)
	wg.Wait()
	fmt.Println("Finished!")
}
