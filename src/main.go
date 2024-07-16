package main

import (
	"context"

	"orb/runner/src/api/crawler"
	"orb/runner/src/api/meta"
	"orb/runner/src/config"
	"orb/runner/src/log"
	"orb/runner/src/runner"
)

func main() {
	switch config.C.Api.Type {
	case config.CrawlerApi:
		hooks := crawler.CrawlerApiHooks{}
		instance, err := runner.New[
			crawler.CrawlingResult, crawler.CrawlerResponse,
		](&hooks)
		if err != nil {
			log.S.Fatalw("Error in runner initialization", "error", err)
		}
		instance.Run(context.Background())
	case config.MetaApi:
		hooks := meta.MetaApiHooks{}
		instance, err := runner.New[
			meta.VerificationResult, meta.VerifyResponse,
		](&hooks)
		if err != nil {
			log.S.Fatalw("Error in runner initialization", "error", err)
		}
		instance.Run(context.Background())
	default:
		log.S.Panicw("Unexpected API type", "input_value", config.C.Api.Type)
	}
}
