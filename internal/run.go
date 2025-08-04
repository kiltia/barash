package internal

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

	"go.uber.org/zap"
)

const (
	APINameCrawler string = "crawler"
	APINameMeta    string = "meta"
)

func RunApplication(cfg *config.Config) {
	// application will run using this context
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()
	wg := sync.WaitGroup{}
	log.Init(cfg.Log)

	switch cfg.API.Name {
	case APINameCrawler:
		zap.S().Debug("Initializing a new Crawler instance")
		queryBuilder := crawler.CrawlerQueryBuilder{
			BatchSize:          cfg.Run.SelectionBatchSize,
			Mode:               cfg.Run.Mode,
			LastId:             0,
			SelectionTableName: cfg.Run.SelectionTableName,
		}
		queryBuilder.ResetState()
		instance, err := runner.New[
			crawler.CrawlerResult, crawler.CrawlerResponse,
		](cfg, &queryBuilder)
		if err != nil {
			zap.S().Fatalw("Error in runner initialization: ", "error", err)
		}
		instance.Run(ctx, &wg)
	case APINameMeta:
		queryBuilder := meta.VerifyQueryBuilder{
			Interval:           cfg.Run.Freshness,
			Limit:              cfg.Run.SelectionBatchSize,
			Mode:               cfg.Run.Mode,
			SelectionTableName: cfg.Run.SelectionTableName,
		}
		queryBuilder.ResetState()
		instance, err := runner.New[
			meta.VerifyResult, meta.VerifyResponse,
		](cfg, &queryBuilder)
		if err != nil {
			zap.S().Fatal("Error in runner initialization: ", err)
		}
		instance.Run(ctx, &wg)
	default:
		zap.S().Panic("Unexpected API name: ", cfg.API.Name)
	}

	timeout := cfg.Timeouts.ShutdownTimeout

	done := make(chan struct{})

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		zap.S().
			Info("Shutting down gracefully, Ctrl+C to force. Timeout: ", timeout)
		select {
		case <-time.After(timeout):
			zap.S().Info("Shutdown timeout reached, forcefully shutting down.")
		case <-done:
			zap.S().Info("Shutdown completed.")
		}
	case <-done:
		zap.S().Info("Writer is stopped. Shutting down the application")
	}
}
