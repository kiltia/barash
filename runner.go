package barash

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/kiltia/barash/config"

	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"
	"resty.dev/v3"
)

type ContextKey int

const (
	ContextKeyFetcherNum ContextKey = iota
)

var _ Sink[StoredResult] = &Clickhouse[StoredResult, StoredParams]{}

type Runner[S StoredResult, R Response[S, P], P StoredParams, Q QueryBuilder[P]] struct {
	sinks          []Sink[S]
	src            Source[P]
	httpClient     *resty.Client
	cfg            *config.Config
	circuitBreaker *gobreaker.CircuitBreaker[*resty.Response]
	queryBuilder   Q

	selectSQL string
}

func New[
	S StoredResult,
	R Response[S, P],
	P StoredParams,
	Q QueryBuilder[P],
](
	cfg *config.Config,
	qb Q,
) (*Runner[S, R, P, Q], error) {
	chSource, version, err := NewClickHouseClient[S, P](
		cfg.Provider.Source.Credentials,
	)
	if err != nil {
		return nil, err
	}

	chSink, version, err := NewClickHouseClient[S, P](
		cfg.Writer.Sink.Credentials,
	)
	if err != nil {
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

	selectSQL, err := os.ReadFile(cfg.Provider.SelectSQLPath)
	if err != nil {
		return nil, fmt.Errorf("reading select sql statement: %w", err)
	}

	runner := Runner[S, R, P, Q]{
		httpClient: httpClient,
		src:        chSource,
		cfg:        cfg,
		// TODO(nrydanov): Remove hardcode when others backends become available
		sinks:        []Sink[S]{chSink},
		selectSQL:    string(selectSQL),
		queryBuilder: qb,
	}

	runner.circuitBreaker = gobreaker.NewCircuitBreaker[*resty.Response](
		gobreaker.Settings{
			Name:        "outgoing_requests",
			MaxRequests: cfg.CircuitBreaker.MaxRequests,
			Interval:    cfg.CircuitBreaker.Interval,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				if cfg.CircuitBreaker.Enabled {
					tooManyTotal := counts.TotalFailures > cfg.CircuitBreaker.TotalFailurePerInterval
					tooManyConsecutive := counts.ConsecutiveFailures > cfg.CircuitBreaker.ConsecutiveFailure
					return tooManyTotal || tooManyConsecutive
				} else {
					return false
				}
			},
		})
	return &runner, nil
}

// Run the runner's job within a given context.
func (r *Runner[S, R, P, Q]) Run(
	ctx context.Context,
	globalWg *sync.WaitGroup,
) {
	// initialize storage in two-table mode
	err := r.initTable(ctx)
	if err != nil {
		zap.S().
			Warnw("one or more table creation scripts have failed", "error", err)
	} else {
		zap.S().Infow("successfully initialized table for the Runner results")
	}

	tasks := r.startProvider(globalWg, ctx)
	results := r.startFetchers(globalWg, ctx, tasks)
	r.startWriter(globalWg, results)
}

func (r *Runner[S, R, P, Q]) initTable(
	ctx context.Context,
) error {
	if r.cfg.Mode == config.ContinuousMode {
		zap.S().
			Infow("running in continuous mode, skipping table initialization")
		return nil
	}
	var errs []error
	for _, sink := range r.sinks {
		err := sink.InitTable(
			ctx,
		)
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
