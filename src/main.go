package main

import (
	"context"

	metaapi "orb/runner/src/api/meta"
	"orb/runner/src/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// TODO(nrydanov): Add support for YAML configuration and choose generics
	// based on this value
	metaRunner := NewRunner[
		metaapi.VerificationResult,
		metaapi.VerificationResponse,
	](cfg)
	if metaRunner == nil {
		return
	}

	metaRunner.Run(context.Background())
}
