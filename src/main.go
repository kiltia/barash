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
	start := time.Now()
	runner.Run()
	elapsed := time.Since(start)
	fmt.Printf("Binomial took %s", elapsed)
}
