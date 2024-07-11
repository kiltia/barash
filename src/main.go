package main

import (
	"context"

	"orb/runner/src/api"
	"orb/runner/src/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	// TODO(nrydanov): Add support for YAML configuration and choose generics
	// based on this value
	runner := NewRunner[api.VerificationResult, api.VerificationResponse](cfg)
	if runner == nil {
		return
	}
	runner.Run(context.Background())
}
