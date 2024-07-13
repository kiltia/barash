package main

import (
	"context"

	metaapi "orb/runner/src/api/meta"
	"orb/runner/src/runner"
)

func main() {
	// TODO(nrydanov): Add support for YAML configuration and choose generics
	// based on this value
	metaRunner := runner.New[
		metaapi.VerificationResult,
		metaapi.VerificationResponse,
	]()
	if metaRunner == nil {
		return
	}

	service := metaapi.MetaApi{}
	metaRunner.Run(context.Background(), &service)
}
