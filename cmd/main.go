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
	wg := sync.WaitGroup{}
	logObject := log.L().Tag(log.LogTagMain)

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
		instance.Run(ctx, &wg)
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
		instance.Run(ctx, &wg)
	default:
		log.S.Panic(
			"Unexpected API name",
			logObject.Add("input_value", config.C.Api.Name),
		)
	}

	timeout := time.Duration(config.C.Run.ShutdownTimeout) * time.Second

	done := make(chan struct{})

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		log.S.Info(
			"Shutting down gracefully, Ctrl+C to force.",
			logObject.Add("timeout", timeout),
		)
		cancel()
		select {
		case <-time.After(timeout):
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
	case <-done:
		log.S.Info(
			"Writer is stopped. Shutting down the application",
			logObject,
		)
	}

}
