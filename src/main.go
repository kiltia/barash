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
		queryBuilder := crawler.CrawlerQueryBuilder{
			BatchSize: config.C.Run.BatchSize,
		}
        queryBuilder.ResetState()
		instance, err := runner.New[
			crawler.CrawlingResult, crawler.CrawlerResponse,
		](&hooks, &queryBuilder)
		if err != nil {
			log.S.Fatalw("Error in runner initialization", "error", err)
		}
		instance.Run(context.Background())
	case config.MetaApi:
		hooks := meta.MetaApiHooks{}
		queryBuilder := meta.MetaQueryBuilder{
			Offset: config.C.Run.TableData.Freshness,
			Limit:  config.C.Run.BatchSize,
		}
		queryBuilder.ResetState()
		instance, err := runner.New[
			meta.MetaResult, meta.MetaResponse,
		](&hooks, &queryBuilder)
		if err != nil {
			log.S.Fatalw("Error in runner initialization", "error", err)
		}
		instance.Run(context.Background())
	default:
		log.S.Panicw("Unexpected API type", "input_value", config.C.Api.Type)
	}
}
