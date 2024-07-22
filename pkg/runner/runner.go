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
	workerTimeout    time.Duration
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
		workerTimeout: time.Duration(
			config.C.Timeouts.GoroutineTimeout,
		) * time.Second,
		hooks:        hs,
		queryBuilder: qb,
	}
	return &runner, nil
}

// Run the runner's job within a given context.
func (r *Runner[S, R, P, Q]) Run(ctx context.Context) {
	logObject := log.L().Tag(log.LogTagRunner)

	// initialize storage in two-table mode
	r.initTable(ctx)
	defer log.S.Debug("The runner's main routine is completed", logObject)

	var remainder []S
	writerTasks := make(chan []S, 1)
	defer close(writerTasks)

	go func() {
		for task := range writerTasks {
			select {
			case <-ctx.Done():
				return
			default:
				// save results to the database
				err := r.write(ctx, task)
				if err != nil {
					log.S.Error(
						"Failed to save processed batch to the database",
						logObject.Error(err),
					)
				}
			}
		}
	}()

	// main runner's loop
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// batch processing start time
			timestamp := time.Now()

			// fetch request parameters from the database
			params, err := r.fetchParams(ctx)
			if err != nil {
				log.S.Error(
					"Failed to fetch request parameters from the database",
					logObject.Error(err),
				)
				continue
			}

			// check that the set is not empty
			if len(params) == 0 {
				log.S.Info(
					"Runner has nothing to do, soon entering standby mode",
					log.L().
						Tag(log.LogTagRunner).
						Add("sleep_time", config.C.Run.SleepTime),
				)
				if len(remainder) > 0 {
					log.S.Debug(
						"Sending results to the writer",
						logObject,
					)
					writerTasks <- remainder
				}
				remainder = []S{}
				err = r.standby(ctx)
				if err != nil {
					return // context is cancelled
				}
                r.queryBuilder.ResetState()
				continue // try again
			}

			// stride over records in the database
			r.queryBuilder.UpdateState(params)

			// create requests using runner's configuration
			// and parameters from the database
			requests := r.formRequests(params)

			// perform requests, gather results
			var processed [][]S
			remainder, processed = r.fetch(
				ctx,
				requests,
				remainder,
				writerTasks,
			)

			// perform quality control checks
			err = r.qualityControl(ctx, processed, timestamp)
			if err != nil {
				return // context is cancelled
			}
		}
	}
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
