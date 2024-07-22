package main

import (
	"context"
	"os/signal"
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

	// channel to signal when the application has fully stopped
	done := make(chan bool)

	go func() {
		defer close(done)

		switch config.C.Api.Name {
		case ApiNameCrawler:
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
				log.S.Fatal(
					"Error in runner initialization",
					log.L().Tag(log.LogTagRunner).Error(err),
				)
			}
			instance.Run(ctx)
		case ApiNameMeta:
			hooks := meta.VerifyApiHooks{}
			queryBuilder := meta.VerifyQueryBuilder{
				DayInterval:    config.C.Run.Freshness,
				Limit:          config.C.Run.BatchSize,
				Mode:           config.C.Run.Mode,
				StartTimestamp: time.Now(),
				CurrentTag:     config.C.Run.Tag,
			}
			queryBuilder.ResetState()
			instance, err := runner.New[
				meta.VerifyResult, meta.VerifyResponse,
			](&hooks, &queryBuilder)
			if err != nil {
				log.S.Fatal(
					"Error in runner initialization",
					log.L().Tag(log.LogTagRunner).Error(err),
				)
			}
			instance.Run(ctx)
		default:
			log.S.Panic(
				"Unexpected API name",
				log.L().Tag(log.LogTagRunner).
					Add("input_value", config.C.Api.Name),
			)
		}
	}()

	<-ctx.Done() // wait for the termination signal
	log.S.Info(
		"Shutting down gracefully, Ctrl+C to force.",
		log.L().Tag(log.LogTagRunner).
			Add("timeout", 10),
	)
	cancel() // restore normal signal behavior

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		log.S.Debug(
			"Timeout reached, forcing shutdown.",
			log.L().Tag(log.LogTagRunner),
		)
	}
}
