package main

import (
	"context"

	"orb/runner/src/api"
)

func main() {
	config := NewRunnerConfig()
	if config == nil {
		return
	}
	// TODO(nrydanov): Add support for YAML configuration and choose generics
	// based on this value
	runner := NewRunner[api.VerificationResult, api.VerificationResponse](
		*config,
	)
	if runner == nil {
		return
	}
	runner.Run(context.Background())
}
