package runner

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"orb/runner/pkg/config"

	"github.com/avast/retry-go/v4"
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

	fetcherCh := make(chan ServiceRequest[P], 2*r.cfg.Run.SelectionBatchSize)
	writerCh := make(chan S, 2*r.cfg.Run.InsertionBatchSize+1)
	wg := sync.WaitGroup{}

	fetcherCnt := atomic.Int32{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		r.dataProvider(ctx, fetcherCh)
	}()

	go func() {
		for {
			time.Sleep(time.Second * 10)
			zap.S().Infow(
				"Warm up is in progress",
				"fetcher_cnt", fetcherCnt.Load(),
			)
			if fetcherCnt.Load() >= int32(r.cfg.Run.MaxFetcherWorkers) {
				zap.S().Infow("Warm up has ended")
				return
			}
		}
	}()

	wg.Add(r.cfg.Run.MaxFetcherWorkers)
	for i := range r.cfg.Run.MaxFetcherWorkers {
		var rnd time.Duration
		if i < r.cfg.Run.MinFetcherWorkers {
			rnd = 0
		} else {
			rnd = time.Duration(rand.IntN(int(r.cfg.Run.WarmupTime.Seconds())+1)) * time.Second
		}
		go func() {
			defer wg.Done()
			time.Sleep(rnd)
			fetcherCnt.Add(1)
			r.fetcher(ctx, fetcherCh, writerCh, i)
		}()
	}

	globalWg.Add(1)
	go func() {
		defer globalWg.Done()
		r.writer(ctx, writerCh, &wg)
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
