package runner

import (
	"context"
	"sync"
	"time"

	"github.com/kiltia/runner/pkg/config"

	"go.uber.org/zap"
)

func (r *Runner[S, R, P, Q]) startProvider(
	ctx context.Context,
	globalWg *sync.WaitGroup,
) chan ServiceRequest[P] {
	out := make(chan ServiceRequest[P], 2*r.cfg.Run.SelectionBatchSize)
	globalWg.Add(1)

	go func() {
		defer close(out)
		defer globalWg.Done()
		for {
			select {
			case <-ctx.Done():
				zap.S().Info("Data provider has been stopped")
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
						len(out) == 0 {
						zap.S().Infow("All data is processed, exiting")
						return
					} else {
						r.queryBuilder.ResetState()
						zap.S().Infow(
							"The data provider has nothing to do, entering standby mode",
							"sleep_time", r.cfg.Run.SleepTime,
							"tasks_left", len(out),
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
						select {
						case <-ctx.Done():
							zap.S().Info("Data provider has been stopped")
							return
						case out <- r:

						}
					}
					zap.S().Debug("A batch was completely sent to fetchers")
				}
			}
		}
	}()

	return out
}
