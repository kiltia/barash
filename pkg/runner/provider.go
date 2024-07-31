package runner

import (
	"context"

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

	for {
		select {
		case <-ctx.Done():
			return
		default:
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
			} else {
				r.queryBuilder.UpdateState(params)

				// create requests using runner's configuration
				// and parameters from the database
				requests := r.formRequests(params)
				for _, r := range requests {
					fetchTasks <- r
				}
			}
			log.S.Debug("A batch was completely sent to fetchers", logObject)
		}
	}
}
