package main

import (
	"context"

	"orb/runner/internal/crawler"
	"orb/runner/internal/meta"
	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	"orb/runner/pkg/runner"
)

func main() {
	switch config.C.Api.Type {
	case config.CrawlerApi:
		hooks := crawler.CrawlerApiHooks{}
		queryBuilder := crawler.CrawlerQueryBuilder{
			BatchSize: config.C.Run.BatchSize,
			Mode:      config.C.Run.Mode,
			Offset:    0,
		}
		queryBuilder.ResetState()
		instance, err := runner.New[
			crawler.CrawlerResult, crawler.CrawlerResponse,
		](&hooks, &queryBuilder)
		if err != nil {
			log.S.Fatalw("Error in runner initialization", "error", err)
		}
		instance.Run(context.Background())
	case config.MetaApi:
		hooks := meta.VerifyApiHooks{}
		queryBuilder := meta.VerifyQueryBuilder{
			Offset: config.C.Run.Freshness,
			Limit:  config.C.Run.BatchSize,
			Mode:   config.C.Run.Mode,
		}
		queryBuilder.ResetState()
		instance, err := runner.New[
			meta.VerifyResult, meta.VerifyResponse,
		](&hooks, &queryBuilder)
		if err != nil {
			log.S.Fatalw("Error in runner initialization", "error", err)
		}
		instance.Run(context.Background())
	default:
		log.S.Panicw("Unexpected API type", "input_value", config.C.Api.Type)
	}
}
