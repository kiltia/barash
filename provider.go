package barash

import (
	"context"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/kiltia/barash/config"

	"go.uber.org/zap"
)

func (r *Runner[S, R, P, Q]) startProvider(
	wg *sync.WaitGroup,
	ctx context.Context,
) chan ServiceRequest[P] {
	out := make(chan ServiceRequest[P], 2*r.cfg.Provider.SelectionBatchSize)

	var p *P
	var mutator BodyMutator
	if _, ok := any(p).(IncludeBodyFromFile); ok {
		mutator = NewBodyMutator(r.cfg.API.BodyFilePath)
	}

	var requestsCh chan ServiceRequest[P]
	wg.Go(func() {
		defer close(out)
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
				requestsCh, err = r.gatherRequests(ctx, mutator)
				if err != nil {
					zap.S().Errorw("gathering requests", "error", err)
					return
				}

				// If there're more tasks to be completed, we continue
				if requestsCh != nil {
					continue
				}

				// Otherwise, depending on the mode, we either exit or enter standby mode
				switch r.cfg.Mode {
				case config.TwoTableMode:
					zap.S().Infow("data is processed, exiting")
					return
				case config.ContinuousMode:
					r.queryBuilder.ResetState()
					zap.S().Infow(
						"provider has nothing to do, entering standby mode",
						"sleep_time", r.cfg.Provider.SleepTime,
						"tasks_left", len(out),
					)
					select {
					case <-ctx.Done():
						return
					case <-time.After(r.cfg.Provider.SleepTime):
						continue
					}
				}
			}
		}
	})

	return out
}

func (r *Runner[S, R, P, Q]) gatherRequests(
	ctx context.Context,
	mutator BodyMutator,
) (chan ServiceRequest[P], error) {
	zap.S().Debug("trying to get more tasks for fetchers")
	params, err := r.fetchParams(
		ctx,
	)
	for i := range params {
		if p, ok := any(&params[i]).(IncludeBodyFromFile); ok {
			mutator.Mutate(p)
		}
	}
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

// Forms requests using runner's configuration ([api] section in the config
// file) and a set of request parameters fetched from the database.
func (r *Runner[S, R, P, Q]) createRequestStream(
	params []P,
) chan ServiceRequest[P] {
	ch := make(chan ServiceRequest[P], len(params))
	for i := range params {
		p := &params[i]
		ch <- ServiceRequest[P]{
			Host:        r.cfg.API.Host,
			Port:        r.cfg.API.Port,
			Endpoint:    r.cfg.API.Endpoint,
			Scheme:      r.cfg.API.Scheme,
			Method:      r.cfg.API.Method,
			Params:      *p,
			ExtraParams: r.cfg.API.ExtraParams,
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
				params, err = r.src.GetNextBatch(
					ctx,
					r.selectSQL,
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
				r.cfg.Provider.SelectRetries,
			)+1,
		),
	)
	return params, err
}
