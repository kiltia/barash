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
	paramsList := loadVerifyParamsFromCSV("./dnb_tmp_wv_sample.csv")
	var verifyGetRequestList []VerifyGetRequest
	for i := 0; i < 20; i++ {
		verifyGetRequest := NewVerifyGetRequest("http://127.0.0.1", "8081", "/verify", paramsList[i])
		verifyGetRequestList = append(verifyGetRequestList, *verifyGetRequest)
	}
	start := time.Now()
	runner.Run(6, 1, verifyGetRequestList)
	elapsed := time.Since(start)
	fmt.Printf("Binomial took %s", elapsed)
}
