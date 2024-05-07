package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

func SendGetRequest(url string) string {
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

func producer(tasks *chan string, results *chan string, wg *sync.WaitGroup) {
	for len(*tasks) != 0 {
		task, ok := <-*tasks
		if !ok {
			break
		}
		result := SendGetRequest(task)
		*results <- result
	}
	fmt.Println("Producer is finished!")
	wg.Done()
}

func consumer(results *chan string, numTasks *int, wg *sync.WaitGroup) {
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

func Run(producerSize int, consumerSize int, verifyGetRequestList []VerifyGetRequest) {
	tasks := make(chan string, len(verifyGetRequestList))
	var wg sync.WaitGroup
	for _, t := range verifyGetRequestList {
		task, _ := t.CreateVerifyGetRequestLink()
		tasks <- task
	}
	results := make(chan string, consumerSize)
	for i := 0; i < producerSize; i++ {
		wg.Add(1)
		go producer(&tasks, &results, &wg)
	}
	var numTasks int = len(verifyGetRequestList)
	for i := 0; i < consumerSize; i++ {
		wg.Add(1)
		go consumer(&results, &numTasks, &wg)
	}
	defer close(results)
	defer close(tasks)
	wg.Wait()
	fmt.Println("Finished!")
}

func main() {
	var url = "https://apple.com/"
	var locState = "California"
	var name = "Apple Inc"
	var locCity = "Cupertino"
	verifyParams := VerifyParams{Url: url, LocState: &locState, Name: &name, LocCity: &locCity}
	/*
		verifyGetRequest := NewVerifyGetRequest("http://127.0.0.1", "8081", "/verify", verifyParams)
		var verifyGetRequestList []VerifyGetRequest
		for i := 1; i < 2; i++ {
			verifyGetRequestList = append(verifyGetRequestList, verifyGetRequest)
		}
		start := time.Now()
		Run(6, 1, verifyGetRequestList)
		elapsed := time.Since(start)
		fmt.Printf("Binomial took %s", elapsed)
	*/
	client, err := NewClickHouseClient(*NewClickHouseConfig())
	if err != nil {
		return
	}
	client.AsyncInsertBatch([]VerifyParams{verifyParams, verifyParams}, []float64{0.01, 0.2})
}
