package main

import (
	"fmt"
	"time"
)

func main() {

	config := NewRunnerConfig()
	runner := NewRunner(*config)
	if runner == nil {
		return
	}
	var url = "https://apple.com/"
	var locState = "California"
	var name = "Apple Inc"
	var locCity = "Cupertino"
	verifyParams := VerifyParams{Url: url, LocState: &locState, Name: &name, LocCity: &locCity}

	verifyGetRequest := NewVerifyGetRequest("http://127.0.0.1", "8081", "/verify", verifyParams)
	var verifyGetRequestList []VerifyGetRequest
	for i := 1; i < 8; i++ {
		verifyGetRequestList = append(verifyGetRequestList, *verifyGetRequest)
	}
	start := time.Now()
	runner.Run(6, 1, verifyGetRequestList)
	elapsed := time.Since(start)
	fmt.Printf("Binomial took %s", elapsed)

}
