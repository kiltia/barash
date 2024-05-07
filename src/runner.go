package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

type Runner struct {
	clickHouseClient ClickHouseClient
	verifierCreds    VerifierConfig
}

func NewRunner(config RunnerConfig) *Runner {
	clickHouseClient, err := NewClickHouseClient(config.ClickHouseConfig)
	if err != nil {
		return nil
	}
	var verifierCreds = config.VerifierConfig
	return &Runner{clickHouseClient: *clickHouseClient, verifierCreds: verifierCreds}
}

func (runner Runner) SendGetRequest(url string) string {
	response, err := http.Get(url)
	fmt.Println(response.StatusCode)
	if err != nil {
		fmt.Printf("Gotten error %s", err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Gotten error %s", err)
	}
	return string(body)
}

func (runner Runner) producer(tasks *chan string, results *chan string, wg *sync.WaitGroup) {
	for len(*tasks) != 0 {
		task, ok := <-*tasks
		if !ok {
			break
		}
		result := runner.SendGetRequest(task)
		*results <- result
	}
	fmt.Println("Producer is finished!")
	wg.Done()
}

func (runner Runner) consumer(results *chan string, numTasks *int, wg *sync.WaitGroup) {
	for *numTasks != 0 {
		if len(*results) != 0 {
			result, ok := <-*results
			if !ok {
				break
			}
			fmt.Println(result[:3])
			*numTasks = *numTasks - 1
			fmt.Println(*numTasks)
		}
	}
	fmt.Println("Consumer is finished!")
	wg.Done()
}

func (runner Runner) Run(producerSize int, consumerSize int, verifyGetRequestList []VerifyGetRequest) {
	tasks := make(chan string, len(verifyGetRequestList))
	var wg sync.WaitGroup
	for _, t := range verifyGetRequestList {
		task, _ := t.CreateVerifyGetRequestLink()
		tasks <- task
	}
	results := make(chan string, consumerSize)
	for i := 0; i < producerSize; i++ {
		wg.Add(1)
		go runner.producer(&tasks, &results, &wg)
	}
	var numTasks int = len(verifyGetRequestList)
	for i := 0; i < consumerSize; i++ {
		wg.Add(1)
		go runner.consumer(&results, &numTasks, &wg)
	}
	defer close(results)
	defer close(tasks)
	wg.Wait()
	fmt.Println("Finished!")
}
