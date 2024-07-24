package runner

import (
	"context"
	"fmt"
	"time"

	"orb/runner/pkg/config"
	dbclient "orb/runner/pkg/db/clients"
	"orb/runner/pkg/log"
	"orb/runner/pkg/runner/hooks"
	ri "orb/runner/pkg/runner/interface"
	rr "orb/runner/pkg/runner/request"

	"github.com/avast/retry-go/v4"
	"github.com/go-resty/resty/v2"
)

type Runner[S ri.StoredValue, R ri.Response[S, P], P ri.StoredParams, Q ri.QueryBuilder[S, P]] struct {
	clickHouseClient dbclient.ClickHouseClient[S, P, Q]
	httpClient       *resty.Client
	hooks            hooks.Hooks[S]
	queryBuilder     Q
}

func New[
	S ri.StoredValue,
	R ri.Response[S, P],
	P ri.StoredParams,
	Q ri.QueryBuilder[S, P],
](hs hooks.Hooks[S], qb Q) (*Runner[S, R, P, Q], error) {
	logObject := log.L().Tag(log.LogTagRunner)

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
			logObject.Error(err),
		)
		return nil, err
	}

	log.S.Info(
		"Created a new ClickHouse client",
		logObject.Add("version", fmt.Sprintf("%v", version)),
	)

	runner := Runner[S, R, P, Q]{
		clickHouseClient: *clickHouseClient,
		httpClient:       initHttpClient(),
		hooks:            hs,
		queryBuilder:     qb,
	}
	return &runner, nil
}

// Run the runner's job within a given context.
func (r *Runner[S, R, P, Q]) Run(ctx context.Context) {
	logObject := log.L().Tag(log.LogTagRunner)

	// initialize storage in two-table mode
	r.initTable(ctx)
	defer log.S.Debug("The runner's main routine is completed", logObject)

	fetcherCh := make(chan rr.GetRequest[P], 2*config.C.Run.BatchSize)
	writerCh := make(chan S, config.C.Run.BatchSize)
	qcChannel := make(chan []S, 1)
	nothingLeft := make(chan bool)
	standbyChannels := make([]chan bool, config.C.Run.FetcherWorkers)
	go r.dataProvider(ctx, fetcherCh, nothingLeft)

	for i := range config.C.Run.FetcherWorkers {
		standbyChannels[i] = make(chan bool)
		go r.fetcher(ctx, fetcherCh, writerCh, standbyChannels[i], i)
	}

	go r.writer(ctx, writerCh, qcChannel, nothingLeft)

	go r.qualityControl(ctx, qcChannel, time.Now(), &standbyChannels)
}

// Fetch a new set of request parameters from the database.
func (r *Runner[S, R, P, Q]) fetchParams(
	ctx context.Context,
) (params []P, err error) {
	logObject := log.L().Tag(log.LogTagRunner)

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
		retry.Attempts(uint(config.C.SelectRetries.NumRetries)+1),
	)
	return params, err
}

// Forms requests using runner's configuration ([api] section in the config
// file) and a set of request parameters fetched from the database.
func (r *Runner[S, R, P, Q]) formRequests(params []P) (
	requests []rr.GetRequest[P],
) {
	logObject := log.L().Tag(log.LogTagRunner)

	log.S.Debug(
		"Creating requests for the fetching process",
		logObject,
	)
	for _, params := range params {
		requests = append(requests, rr.GetRequest[P]{
			Host:   config.C.Api.Host,
			Port:   config.C.Api.Port,
			Method: config.C.Api.Method,
			Params: params,
		})
	}
	return requests
}
