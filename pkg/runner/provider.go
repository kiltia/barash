package runner

import (
	"context"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
)

func (r *Runner[S, R, P, Q]) dataProvider(
	ctx context.Context,
	fetchTasks chan ServiceRequest[P],
) {
	logObject := log.L().Tag(log.LogTagDataProvider)
	defer close(fetchTasks)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.S.Debug(
				"Trying to get more tasks for fetchers",
				logObject,
			)
			params, err := r.fetchParams(
				ctx,
			)
			r.queryBuilder.UpdateState(params)
			if err != nil {
				log.S.Error(
					"Failed to fetch request parameters from the database",
					logObject.Error(
						err,
					),
				)
				return
			}

			if len(params) == 0 {
				if config.C.Run.Mode == config.TwoTableMode &&
					len(fetchTasks) == 0 {
					log.S.Info("All data is processed, exiting", logObject)
					return
				} else {
					r.queryBuilder.ResetState()
					log.S.Info(
						"The data provider has nothing to do, entering standby mode",
						logObject.
							Add("sleep_time", config.C.Run.SleepTime).
							Add("tasks_left", len(fetchTasks)),
					)
					select {
					case <-ctx.Done():
						return
					case <-time.After(config.C.Run.SleepTime):
						continue
					}
				}
			} else {
				requests := r.formRequests(params)
				for _, r := range requests {
					fetchTasks <- r
				}
				log.S.Debug(
					"A batch was completely sent to fetchers",
					logObject,
				)
			}
		}
	}
}
