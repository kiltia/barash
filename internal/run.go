package internal

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kiltia/runner/internal/crawler"
	"github.com/kiltia/runner/internal/meta"
	"github.com/kiltia/runner/pkg/config"
	"github.com/kiltia/runner/pkg/log"
	"github.com/kiltia/runner/pkg/runner"

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
		queryBuilder := crawler.CrawlerQueryBuilder{
			BatchSize:          cfg.Storage.SelectionBatchSize,
			Mode:               cfg.Mode,
			LastID:             0,
			SelectionTableName: cfg.Storage.SelectionTableName,
		}
		queryBuilder.ResetState()
		instance, err := runner.New[
			crawler.CrawlerResult, crawler.CrawlerResponse,
		](cfg, &queryBuilder)
		if err != nil {
			zap.S().Fatal(fmt.Errorf("runner initialization: %w", err))
		}
		instance.Run(ctx, &wg)
	case APINameMeta:
		queryBuilder := meta.VerifyQueryBuilder{
			Interval:           cfg.ContinuousMode.Freshness,
			Limit:              cfg.Storage.SelectionBatchSize,
			Mode:               cfg.Mode,
			SelectionTableName: cfg.Storage.SelectionTableName,
		}
		queryBuilder.ResetState()
		instance, err := runner.New[
			meta.VerifyResult, meta.VerifyResponse,
		](cfg, &queryBuilder)
		if err != nil {
			zap.S().Fatal(fmt.Errorf("runner initialization: %w", err))
		}
		instance.Run(ctx, &wg)
	default:
		zap.S().Panic("unexpected API name: ", cfg.API.Name)
	}
	timeout := cfg.Shutdown.GracePeriod

	done := make(chan struct{})

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		zap.S().
			Infow("shutting down gracefully, ctrl+c to force", "timeout", timeout)
		forceCtx, cancel := signal.NotifyContext(
			context.Background(),
			syscall.SIGINT,
			syscall.SIGTERM,
		)
		defer cancel()
		select {
		case <-time.After(timeout):
			zap.S().Info("timeout reached, forcefully shutting down")
		case <-done:
			zap.S().Info("graceful shutdown completed")
		case <-forceCtx.Done():
			zap.S().Info("ctrl+c pressed, forcefully shutting down")
		}
	case <-done:
		zap.S().Info("all workers are stopped gracefully, exiting")
	}
}
