package runner

import (
	"context"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	rr "orb/runner/pkg/runner/request"
)

func (r *Runner[S, R, P, Q]) dataProvider(
	ctx context.Context,
	fetchTasks chan rr.GetRequest[P],
	nothingLeft chan bool,
) {
	logObject := log.L().Tag(log.LogTagRunner)
	sleepTime := 0 * time.Second
	startTime := time.Now()
	lastTaskCount := config.C.Run.BatchSize
	timeElapsed := 0 * time.Second
	totalTasks := int64(1)
	for {
		log.S.Info(
			"Data provider started a new iteration",
			logObject.
				Add("time_per_request", (sleepTime/time.Duration(config.C.Run.BatchSize)).Seconds()).
				Add("current_sleep_time", sleepTime.Seconds()).
				Add("time_elapsed", timeElapsed.Seconds()).
				Add("task_count", len(fetchTasks)),
		)
		select {
		case <-ctx.Done():
			return
		case <-time.After(sleepTime):
			tasksCount := len(fetchTasks)
			log.S.Debug(
				"Tasks available",
				logObject.Add("tasks", tasksCount),
			)
			timeElapsed = time.Since(startTime)
			totalTasks += int64(lastTaskCount - tasksCount)
			tpr := timeElapsed / time.Duration(totalTasks)
			sleepTime = tpr * time.Duration(config.C.Run.BatchSize)

			if tasksCount < config.C.Run.BatchSize {
				log.S.Debug(
					"Trying to get more tasks for fetchers",
					logObject,
				)
				params, err := r.fetchParams(ctx)
				if err != nil {
					log.S.Error(
						"Failed to fetch request parameters from the database",
						logObject.Error(err),
					)
					return
				}

				if len(params) == 0 && len(fetchTasks) == 0 {
					log.S.Info(
						"Runner has nothing to do, soon entering standby mode",
						log.L().
							Add("sleep_time", config.C.Run.SleepTime),
					)
					nothingLeft <- true
					r.queryBuilder.ResetState()
					err := r.standby(ctx)
					if err != nil {
						return
					}
				}

				// stride over records in the database
				r.queryBuilder.UpdateState(params)

				// create requests using runner's configuration
				// and parameters from the database
				requests := r.formRequests(params)
				for _, r := range requests {
					fetchTasks <- r
				}
			}
			lastTaskCount = len(fetchTasks)
		}
	}
}
