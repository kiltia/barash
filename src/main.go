package main

import (
	"fmt"
	"time"
)

func main() {
	config := NewRunnerConfig()
	if config == nil {
		return
	}
	runner := NewRunner(*config)
	if runner == nil {
		return
	}
	paramsList := loadVerifyParamsFromCSV("./big_5_sample.csv")
	start := time.Now()
	runner.Run(paramsList[:100])
	elapsed := time.Since(start)
	fmt.Printf("Binomial took %s", elapsed)
}
