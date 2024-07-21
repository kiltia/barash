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
	clickHouseClient, version, err := dbclient.NewClickHouseClient[S, P, Q](
		config.C.ClickHouse,
	)
	if err != nil {
		log.S.Errorw(
			"Connection to the ClickHouse database was unsuccessful!",
			"error", err,
			"tag", log.TagClickHouseError,
		)
		return nil, err
	} else {
		log.S.Infow(
			"Connection to the ClickHouse database was successful!",
			"tag", log.TagClickHouseSuccess,
		)
		log.S.Infow(
			fmt.Sprintf("%v", version),
			"tag", log.TagClickHouseSuccess,
		)
	}

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
	// initialize storage in two-table mode
	r.initTable(ctx)
	defer log.S.Info("The Runner's main routine is completed")

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
					log.S.Errorw(
						"Failed to save processed batch to the database",
						"error", err,
						"tag", log.TagClickHouseError,
					)
				}
			}
		}
	}()

	// main runner's loop
	for {
		// batch processing start time
		timestamp := time.Now()

		// fetch request parameters from the database
		params, err := r.fetchParams(ctx)
		if err != nil {
			log.S.Errorw(
				"Failed to fetch request parameters from the database!",
				"error", err,
				"tag", log.TagClickHouseError,
			)
			continue
		}

		// check that the set is not empty
		if len(params) == 0 {
			log.S.Infow(
				"Runner has nothing to do, soon going into standby",
				"sleep_time", config.C.Run.SleepTime,
			)
			if len(remainder) > 0 {
				log.S.Debug("Writing whatever have right now to the database")
				writerTasks <- remainder
			}
			remainder = []S{}
			err = r.standby(ctx)
			if err != nil {
				return // context is cancelled
			}
			continue // try again
		}

		// stride over records in the database
		r.queryBuilder.UpdateState(params)

		// create requests using runner's configuration
		// and parameters from the database
		requests := r.formRequests(params)

		// perform requests, gather results
		var processed [][]S
		remainder, processed = r.fetch(ctx, requests, remainder, writerTasks)

		// perform quality control checks
		totalFails := 0
		for _, batch := range processed {
			report := r.qualityControl(batch, time.Since(timestamp))

			// call user-defined logic (if any)
			r.hooks.AfterBatch(ctx, batch, &report)

			fails := report.TotalFails()
			if fails > 0 {
				log.S.Warnw(
					"Quality control for the current batch was not passed",
					"tag", log.TagQualityControl,
					"fails", fails,
					"details", report,
				)
			}
			totalFails += fails
		}

		if totalFails > 0 {
			log.S.Warnw(
				"Quality control was not passed",
				"tag", log.TagQualityControl,
				"total_fails", totalFails,
			)
			err := r.standby(ctx)
			if err != nil {
				return // context is cancelled
			}
			continue // try again
		}

		log.S.Infow(
			"Quality control has successfully been passed",
			"tag", log.TagQualityControl,
		)
	}
}

// Fetch a new set of request parameters from the database.
func (r *Runner[S, R, P, Q]) fetchParams(
	ctx context.Context,
) (params []P, err error) {
	log.S.Debug("Fetching a new set of request parameters from the database")
	err = retry.Do(
		func() (err error) {
			params, err = r.clickHouseClient.SelectNextBatch(
				ctx,
				r.queryBuilder,
			)
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
	log.S.Debugw(
		"Creating requests for the fetching process",
		"tag", log.TagRunnerDebug,
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
