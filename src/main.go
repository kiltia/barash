package main

import (
	"context"

	"orb/runner/src/api/crawler"
	metaapi "orb/runner/src/api/meta"
	"orb/runner/src/config"
	"orb/runner/src/log"
	"orb/runner/src/runner"
)

func main() {
	// TODO(nrydanov): Add support for YAML configuration and choose generics
	// based on this value
	switch config.C.Api.Type {
	case "crawler":
		hooks := crawler.CrawlerApiHooks{}
		instance, err := runner.New[
			crawler.CrawlingResult, crawler.CrawlerResponse,
		](&hooks)
		if err != nil {
			log.S.Fatalw("Error in runner initialization", "error", err)
		}
		instance.Run(context.Background())
	case "meta":
		hooks := metaapi.MetaApiHooks{}
		instance, err := runner.New[
			metaapi.VerificationResult, metaapi.VerifyResponse,
		](&hooks)
		if err != nil {
			log.S.Fatalw("Error in runner initialization", "error", err)
		}
		instance.Run(context.Background())
	}
}
