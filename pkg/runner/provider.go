package runner

import (
	"context"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/kiltia/runner/pkg/config"

	"go.uber.org/zap"
)

const (
	TaskSendTimeout = 5 * time.Second
)

func (r *Runner[S, R, P, Q]) gatherRequests(
	ctx context.Context,
) (chan ServiceRequest[P], error) {
	zap.S().Debug("trying to get more tasks for fetchers")
	params, err := r.fetchParams(
		ctx,
	)
	r.queryBuilder.UpdateState(params)
	if err != nil {
		return nil, err
	}

	if len(params) > 0 {
		requestsCh := r.createRequestStream(params)
		return requestsCh, nil
	}

	return nil, nil
}

func (r *Runner[S, R, P, Q]) startProvider(
	ctx context.Context,
	globalWg *sync.WaitGroup,
) chan ServiceRequest[P] {
	out := make(chan ServiceRequest[P], 2*r.cfg.Run.SelectionBatchSize)

	var requestsCh chan ServiceRequest[P]
	go func() {
		defer close(out)
		defer globalWg.Done()
		for {
			select {
			case r := <-requestsCh:
				select {
				case out <- r:
					// do nothing
				case <-ctx.Done():
					return
				}
			default:
				var err error
				requestsCh, err = r.gatherRequests(ctx)
				if err != nil {
					zap.S().Errorw("gathering requests", "error", err)
					return
				}

				// If there're more tasks to be completed, we continue
				if requestsCh != nil {
					continue
				}

				// Otherwise, depending on the mode, we either exit or enter standby mode
				switch r.cfg.Run.Mode {
				case config.TwoTableMode:
					zap.S().Infow("data is processed, exiting")
					return
				case config.ContinuousMode:
					r.queryBuilder.ResetState()
					zap.S().Infow(
						"provider has nothing to do, entering standby mode",
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
			}
		}
	}()

	return out
}

// Forms requests using runner's configuration ([api] section in the config
// file) and a set of request parameters fetched from the database.
func (r *Runner[S, R, P, Q]) createRequestStream(
	params []P,
) chan ServiceRequest[P] {
	ch := make(chan ServiceRequest[P], len(params))
	for _, p := range params {
		ch <- ServiceRequest[P]{
			Host:        r.cfg.API.Host,
			Port:        r.cfg.API.Port,
			Endpoint:    r.cfg.API.Endpoint,
			Method:      r.cfg.API.Method,
			Params:      p,
			ExtraParams: r.cfg.Run.ParsedExtraParams,
		}
	}
	return ch
}

// Fetch a new set of request parameters from the database.
func (r *Runner[S, R, P, Q]) fetchParams(
	ctx context.Context,
) (params []P, err error) {
	zap.S().Debug("fetching a new set of request parameters from the database")
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
					zap.S().Errorw(
						"selecting next batch from the database",
						"error", err,
					)
				}
			}
			return err
		},
		retry.Attempts(
			uint(
				r.cfg.SelectRetries.NumRetries,
			)+1,
		),
	)
	return params, err
}
