package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/kiltia/runner/pkg/config"

	"github.com/avast/retry-go/v4"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Runner[S StoredResult, R Response[S, P], P StoredParams, Q QueryBuilder[S, P]] struct {
	clickHouseClient ClickHouseClient[S, P, Q]
	httpClient       *resty.Client
	queryBuilder     Q
	cfg              *config.Config
	stopProvider     chan struct{}
	fetcherCh        chan ServiceRequest[P]
	writerCh         chan S
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
			"Failed to create a new ClickHouse client",
			"error", err,
		)
		return nil, err
	}

	zap.S().Infow(
		"Created a new ClickHouse client",
		"version", fmt.Sprintf("%v", version),
	)

	runner := Runner[S, R, P, Q]{
		clickHouseClient: *clickHouseClient,
		httpClient:       initHTTPClient(cfg.HTTPRetries, cfg.Timeouts),
		queryBuilder:     qb,
		cfg:              cfg,
		stopProvider:     make(chan struct{}),
		fetcherCh: make(
			chan ServiceRequest[P],
			2*cfg.Run.SelectionBatchSize,
		),
		writerCh: make(chan S, 2*cfg.Run.InsertionBatchSize+1),
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
		zap.S().Info("Writer has been stopped")
	}()
}

// Fetch a new set of request parameters from the database.
func (r *Runner[S, R, P, Q]) fetchParams(
	ctx context.Context,
) (params []P, err error) {
	zap.S().Debug("Fetching a new set of request parameters from the database")
	err = retry.Do(
		func() (err error) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				params, err = r.clickHouseClient.SelectNextBatch(
					ctx,
					r.queryBuilder,
				)
				if err != nil {
					zap.S().Errorw(
						"Failed to select the next batch from the database",
						"error", err,
					)
				}
			}
			return err
		},
		retry.Attempts(
			uint(
				r.cfg.SelectRetries.NumRetries,
			)+1,
		),
	)
	return params, err
}

// Forms requests using runner's configuration ([api] section in the config
// file) and a set of request parameters fetched from the database.
func (r *Runner[S, R, P, Q]) formRequests(
	params []P,
) (
	requests []ServiceRequest[P],
) {
	zap.S().Debug("Creating requests for the fetching process")
	for _, params := range params {
		requests = append(
			requests,
			ServiceRequest[P]{
				Host:        r.cfg.API.Host,
				Port:        r.cfg.API.Port,
				Endpoint:    r.cfg.API.Endpoint,
				Method:      r.cfg.API.Method,
				Params:      params,
				ExtraParams: r.cfg.Run.ParsedExtraParams,
			},
		)
	}
	return requests
}
