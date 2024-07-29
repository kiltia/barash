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

	lastTaskCount := config.C.Run.BatchSize
	lastTime := time.Now()

	totalTasks := int64(1)
	totalTime := 0 * time.Second

	updateSleepState := func(tasksCount int) {
		diff := lastTaskCount - tasksCount
		if diff > 0 {
			timeElapsed := time.Since(lastTime)
			totalTime += timeElapsed
			totalTasks += int64(diff)
			tpr := totalTime / time.Duration(totalTasks)
			sleepTime = tpr * time.Duration(config.C.Run.BatchSize)
		}
	}

	for {
		log.S.Info(
			"Data provider started a new iteration",
			logObject.
				Add(
					"time_per_request",
					(sleepTime/time.Duration(config.C.Run.BatchSize)).Seconds(),
				).
				Add("current_sleep_time", sleepTime.Seconds()).
				Add("time_elapsed", totalTime.Seconds()).
				Add("task_count", len(fetchTasks)),
		)
		select {
		case <-ctx.Done():
			return
		case <-time.After(sleepTime):
			tasksCount := len(fetchTasks)

			updateSleepState(tasksCount)

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
					sleepTime = 60 * time.Second
					continue
				} else {
					r.queryBuilder.UpdateState(params)

					// create requests using runner's configuration
					// and parameters from the database
					requests := r.formRequests(params)
					for _, r := range requests {
						fetchTasks <- r
					}
				}
			}

			lastTaskCount = len(fetchTasks)
			lastTime = time.Now()
		}
	}
}
