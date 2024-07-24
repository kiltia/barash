package runner

import (
	"context"
	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	rr "orb/runner/pkg/runner/request"
	"time"
)

func (r *Runner[S, R, P, Q]) dataProvider(ctx context.Context, fetchTasks chan rr.GetRequest[P], nothingLeft chan bool) {
	logObject := log.L().Tag(log.LogTagRunner)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.S.Debug("Tasks available", logObject.Add("tasks", len(fetchTasks)))
            if len(fetchTasks) >= 2 * config.C.Run.BatchSize {
                continue
            }
            log.S.Debug("Trying to get more tasks for fetchers", logObject.Tag(log.LogTagRunner))
			params, err := r.fetchParams(ctx)
			if err != nil {
				log.S.Error(
					"Failed to fetch request parameters from the database",
					logObject.Error(err),
				)
				return
			}

			// check that the set is not empty
			if len(params) == 0 {
				log.S.Info(
					"Runner has nothing to do, soon entering standby mode",
					log.L().
						Tag(log.LogTagRunner).
						Add("sleep_time", config.C.Run.SleepTime),
				)
				nothingLeft <- true
				err = r.standby(ctx)
				if err != nil {
					return // context is cancelled
				}
				r.queryBuilder.ResetState()
				return
				// continue // try again
			}

			// stride over records in the database
			r.queryBuilder.UpdateState(params)

			// create requests using runner's configuration
			// and parameters from the database
			requests := r.formRequests(params)
			for _, r := range requests {
				fetchTasks <- r
			}

            // TODO(nrydanov): Replace with config value
			time.Sleep(10 * time.Second)
		}
	}

}

func (r *Runner[S, R, P, Q]) fetcher(
	ctx context.Context,
	input chan rr.GetRequest[P],
	output chan S,
	fetcherNum int,
) {

	logObject := log.L().Tag(log.LogTagFetching)
	for {
		select {
		case task := <-input:
			storedValues := r.handleFetcherTask(ctx, task, fetcherNum)
			for _, value := range storedValues {
				output <- value
			}
		case <-ctx.Done():
			return
		default:
			log.S.Info("Got nothing to fetch, time to sleep", logObject)
            // TODO(nrydanov): Replace with config value
			time.Sleep(15 * time.Second)
		}
	}
}

func (r *Runner[S, R, P, Q]) writer(ctx context.Context, results chan S, nothingLeft chan bool) {
	logObject := log.L().Tag(log.LogTagWriting)
	var batch []S
	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-results:
			if !ok {
				log.S.Info("Channel is closed", logObject)
			}
			// save results to the database
			batch = append(batch, result)
		case <-nothingLeft:
			log.S.Info("Got \"nothing left\" signal, saving the rest of batch to database", logObject)
			err := r.write(ctx, batch)
			if err != nil {
				log.S.Error(
					"Failed to save processed batch to the database",
					logObject.Error(err),
				)
			}
		default:
			if len(batch) > config.C.Run.BatchSize {
				log.S.Info(
					"Have enough results, saving to the database", logObject,
				)
				err := r.write(ctx, batch)
				if err != nil {
					log.S.Error(
						"Failed to save processed batch to the database",
						logObject.Error(err),
					)
				}
				batch = []S{}

			}
            // TODO(nrydanov): Replace with config value
			time.Sleep(30 * time.Second)

		}
	}
}
