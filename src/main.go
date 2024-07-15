package main

import (
	"context"

	metaapi "orb/runner/src/api/meta"
	"orb/runner/src/log"
	"orb/runner/src/runner"
)

func main() {
	// TODO(nrydanov): Add support for YAML configuration and choose generics
	// based on this value
	hooks := metaapi.MetaApiHooks{}
	metaRunner, err := runner.New[
		metaapi.VerificationResult,
		metaapi.VerifyResponse,
	](&hooks)
	if err != nil {
		log.S.Fatalw("Error in runner initialization", "error", err)
	}

	metaRunner.Run(context.Background())
}
