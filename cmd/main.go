package main

import (
	"context"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"orb/runner/internal/crawler"
	"orb/runner/internal/meta"
	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	"orb/runner/pkg/runner"
)

const (
	ApiNameCrawler string = "crawler"
	ApiNameMeta    string = "meta"
)

func main() {
	// application will run using this context
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()
	wg := new(sync.WaitGroup)
	logObject := log.L().Tag(log.LogTagInit)

	go func() {
		switch config.C.Api.Name {
		case ApiNameCrawler:
			hooks := crawler.CrawlerApiHooks{}
			queryBuilder := crawler.CrawlerQueryBuilder{
				BatchSize: config.C.Run.SelectionBatchSize,
				Mode:      config.C.Run.Mode,
				LastId:    0,
			}
			queryBuilder.ResetState()
			instance, err := runner.New[
				crawler.CrawlerResult, crawler.CrawlerResponse,
			](&hooks, &queryBuilder)
			if err != nil {
				log.S.Fatal(
					"Error in runner initialization",
					logObject.Error(err),
				)
			}
			instance.Run(ctx, wg)
		case ApiNameMeta:
			hooks := meta.VerifyApiHooks{}
			queryBuilder := meta.VerifyQueryBuilder{
				DayInterval:    config.C.Run.Freshness,
				Limit:          config.C.Run.SelectionBatchSize,
				Mode:           config.C.Run.Mode,
				StartTimestamp: time.Now(),
			}
			queryBuilder.ResetState()
			instance, err := runner.New[
				meta.VerifyResult, meta.VerifyResponse,
			](&hooks, &queryBuilder)
			if err != nil {
				log.S.Fatal(
					"Error in runner initialization",
					logObject.Error(err),
				)
			}
			instance.Run(ctx, wg)
		default:
			log.S.Panic(
				"Unexpected API name",
				log.L().Tag(log.LogTagRunner).
					Add("input_value", config.C.Api.Name),
			)
		}
	}()

	<-ctx.Done()
	log.S.Info(
		"Shutting down gracefully, Ctrl+C to force.",
		log.L().Tag(log.LogTagRunner).Add("timeout", 10),
	)
	done := make(chan bool)

	go func() {
        wg.Wait()
		done <- true
	}()

	select {
	case <-time.After(30 * time.Second):
		log.S.Info(
			"Shutdown timeout reached, forcefully shutting down.",
			logObject,
		)
	case <-done:
		log.S.Info(
			"Shutdown completed.",
			logObject,
		)
	}
	cancel()
}
