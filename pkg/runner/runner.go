package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/kiltia/runner/pkg/config"

	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"
	"resty.dev/v3"
)

type Runner[S StoredResult, R Response[S, P], P StoredParams, Q QueryBuilder[S, P]] struct {
	clickHouseClient ClickHouseClient[S, P, Q]
	httpClient       *resty.Client
	queryBuilder     Q
	cfg              *config.Config
	circuitBreaker   *gobreaker.CircuitBreaker[*resty.Response]
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
		cfg.Storage.InsertionTableName,
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
	httpClient := resty.New().
		SetRetryCount(cfg.API.NumRetries).
		SetTimeout(cfg.API.APITimeout).
		SetRetryWaitTime(cfg.API.MinWaitTime).
		SetRetryMaxWaitTime(cfg.API.MaxWaitTime).
		AddRetryConditions(func(r *resty.Response, err error) bool {
			ctx := r.Request.Context()
			fetcherNum := ctx.Value(ContextKeyFetcherNum).(int)
			if r.StatusCode() >= 500 {
				zap.S().
					Debugw(
						"retrying request",
						"fetcher_num",
						fetcherNum,
						"status_code",
						r.StatusCode(),
						"url",
						r.Request.URL,
					)
				return true
			}
			return false
		}).SetLogger(zap.S())

	runner := Runner[S, R, P, Q]{
		clickHouseClient: *clickHouseClient,
		httpClient:       httpClient,
		queryBuilder:     qb,
		cfg:              cfg,
		circuitBreaker: gobreaker.NewCircuitBreaker[*resty.Response](
			gobreaker.Settings{
				Name:     "outgoing_requests",
				Interval: cfg.CircuitBreaker.Interval,
				ReadyToTrip: func(counts gobreaker.Counts) bool {
					tooManyTotal := counts.TotalFailures > cfg.CircuitBreaker.TotalFailurePerInterval
					tooManyConsecutive := counts.ConsecutiveFailures > cfg.CircuitBreaker.ConsecutiveFailure
					return tooManyTotal || tooManyConsecutive
				},
			}),
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

	globalWg.Add(3)
	tasks := r.startProvider(ctx, globalWg)
	results := r.startFetchers(ctx, tasks, globalWg)

	go func() {
		defer globalWg.Done()
		r.writer(results)
		zap.S().Info("writer has been stopped")
	}()
}
