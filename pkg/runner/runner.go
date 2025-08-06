package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/kiltia/runner/pkg/config"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Runner[S StoredResult, R Response[S, P], P StoredParams, Q QueryBuilder[S, P]] struct {
	clickHouseClient ClickHouseClient[S, P, Q]
	httpClient       *resty.Client
	queryBuilder     Q
	cfg              *config.Config
}

func New[
	S StoredResult,
	R Response[S, P],
	P StoredParams,
	Q QueryBuilder[S, P],
](
	cfg *config.Config,
	qb Q,
) (*Runner[S, R, P, Q], error) {
	clickHouseClient, version, err := NewClickHouseClient[S, P, Q](
		cfg.ClickHouse.Host,
		cfg.ClickHouse.Port,
		cfg.ClickHouse.Database,
		cfg.ClickHouse.Username,
		cfg.ClickHouse.Password,
		cfg.Run.InsertionTableName,
		cfg.SelectRetries,
	)
	if err != nil {
		zap.S().Errorw(
			"creating a new clickhouse client",
			"error", err,
		)
		return nil, err
	}

	zap.S().Infow(
		"created a new clickhouse client",
		"version", fmt.Sprintf("%v", version),
	)

	runner := Runner[S, R, P, Q]{
		clickHouseClient: *clickHouseClient,
		httpClient:       initHTTPClient(cfg.HTTPRetries, cfg.Timeouts),
		queryBuilder:     qb,
		cfg:              cfg,
	}
	return &runner, nil
}

// Run the runner's job within a given context.
func (r *Runner[S, R, P, Q]) Run(
	ctx context.Context,
	globalWg *sync.WaitGroup,
) {
	// initialize storage in two-table mode
	r.initTable(ctx)

	tasks := r.startProvider(ctx, globalWg)
	results := r.startFetchers(ctx, tasks, globalWg)

	globalWg.Add(1)
	go func() {
		defer globalWg.Done()
		r.writer(results)
		zap.S().Info("writer has been stopped")
	}()
}
