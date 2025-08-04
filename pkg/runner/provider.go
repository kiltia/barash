package runner

import (
	"context"
	"time"

	"orb/runner/pkg/config"

	"go.uber.org/zap"
)

func (r *Runner[S, R, P, Q]) dataProvider(
	ctx context.Context,
	fetchTasks chan ServiceRequest[P],
) {
	defer close(fetchTasks)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			zap.S().Debug("Trying to get more tasks for fetchers")
			params, err := r.fetchParams(
				ctx,
			)
			r.queryBuilder.UpdateState(params)
			if err != nil {
				zap.S().Errorw(
					"Failed to fetch request parameters from the database",
					"error", err,
				)
				return
			}

			if len(params) == 0 {
				if r.cfg.Run.Mode == config.TwoTableMode &&
					len(fetchTasks) == 0 {
					zap.S().Infow("All data is processed, exiting")
					return
				} else {
					r.queryBuilder.ResetState()
					zap.S().Infow(
						"The data provider has nothing to do, entering standby mode",
						"sleep_time", r.cfg.Run.SleepTime,
						"tasks_left", len(fetchTasks),
					)
					select {
					case <-ctx.Done():
						return
					case <-time.After(r.cfg.Run.SleepTime):
						continue
					}
				}
			} else {
				requests := r.formRequests(params)
				for _, r := range requests {
					fetchTasks <- r
				}
				zap.S().Debug("A batch was completely sent to fetchers")
			}
		}
	}
}
