package runner

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"orb/runner/pkg/config"
	dbclient "orb/runner/pkg/db/clients"
	"orb/runner/pkg/log"
	ri "orb/runner/pkg/runner/interface"
	rr "orb/runner/pkg/runner/request"

	"github.com/avast/retry-go/v4"
	"github.com/go-resty/resty/v2"
)

type Runner[S ri.StoredValue, R ri.Response[S, P], P ri.StoredParams, Q ri.QueryBuilder[S, P]] struct {
	clickHouseClient dbclient.ClickHouseClient[S, P, Q]
	httpClient       *resty.Client
	queryBuilder     Q
}

func New[
	S ri.StoredValue,
	R ri.Response[S, P],
	P ri.StoredParams,
	Q ri.QueryBuilder[S, P],
](
	qb Q,
) (*Runner[S, R, P, Q], error) {
	logObject := log.L().
		Tag(log.LogTagRunner)

	clickHouseClient, version, err := dbclient.NewClickHouseClient[S, P, Q](
		config.C.ClickHouse.Host,
		config.C.ClickHouse.Port,
		config.C.ClickHouse.Database,
		config.C.ClickHouse.Username,
		config.C.ClickHouse.Password,
	)
	if err != nil {
		log.S.Error(
			"Failed to create a new ClickHouse cilent",
			logObject.Error(
				err,
			),
		)
		return nil, err
	}

	log.S.Info(
		"Created a new ClickHouse client",
		logObject.Add(
			"version",
			fmt.Sprintf(
				"%v",
				version,
			),
		),
	)

	runner := Runner[S, R, P, Q]{
		clickHouseClient: *clickHouseClient,
		httpClient:       initHttpClient(),
		queryBuilder:     qb,
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
	logObject := log.L().Tag(log.LogTagRunner)

	fetcherCh := make(chan rr.GetRequest[P], 2*config.C.Run.SelectionBatchSize)
	writerCh := make(chan S, 2*config.C.Run.InsertionBatchSize+1)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		r.dataProvider(ctx, fetcherCh)
	}()

	go func() {
		time.Sleep(time.Duration(config.C.Run.WarmupTime) * time.Second)
		log.S.Info("Warm up has ended", logObject)
	}()

	wg.Add(config.C.Run.MaxFetcherWorkers)
	for i := range config.C.Run.MaxFetcherWorkers {
		var rnd time.Duration
		if i < config.C.Run.MinFetcherWorkers {
			rnd = 0 * time.Second
		} else {
			rnd = time.Duration(rand.IntN(config.C.Run.WarmupTime+1)) * time.Second
		}
		go func() {
			defer wg.Done()
			r.fetcher(ctx, fetcherCh, writerCh, i, rnd)
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
	logObject := log.L().
		Tag(log.LogTagRunner)

	log.S.Debug(
		"Fetching a new set of request parameters from the database",
		logObject,
	)
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
					log.S.Error(
						"Failed to select the next batch from the database",
						logObject.Error(err),
					)
				}
			}
			return err
		},
		retry.Attempts(
			uint(
				config.C.SelectRetries.NumRetries,
			)+1,
		),
	)
	return params, err
}

// Forms requests using runner's configuration ([api] section in the config
// file) and a set of request parameters fetched from the database.
func (r *Runner[S, R, P, Q]) formRequests(
	params []P,
	extraParams map[string]string,
) (
	requests []rr.GetRequest[P],
) {
	logObject := log.L().Tag(log.LogTagRunner)

	log.S.Debug(
		"Creating requests for the fetching process",
		logObject,
	)
	for _, params := range params {
		requests = append(
			requests,
			rr.GetRequest[P]{
				Host:        config.C.Api.Host,
				Port:        config.C.Api.Port,
				Method:      config.C.Api.Endpoint,
				Params:      params,
				ExtraParams: extraParams,
			},
		)
	}
	return requests
}
