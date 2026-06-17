package barash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kiltia/barash/config"
	"github.com/sony/gobreaker/v2"

	"go.uber.org/zap"
	"resty.dev/v3"
)

// breakerPollInterval bounds how often a paused fetcher re-checks the circuit
// breaker state while it is open. Small enough to resume within seconds of the
// breaker's open timeout elapsing, large enough to avoid a hot spin loop.
const breakerPollInterval = 2 * time.Second

func (r *Runner[S, R, P, Q]) fetcher(
	ctx context.Context,
	input <-chan APIRequest[P],
	output chan<- S,
	fetcherNum int,
) {
	logger := zap.S().
		With("fetcher_num", fetcherNum)

	ctx = context.WithValue(ctx, ContextKeyFetcherNum, fetcherNum)

	activeRequests := atomic.Int32{}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// When the breaker is OPEN, stop consuming from the input channel
		// entirely. Leaving tasks unread applies backpressure to the provider
		// (it blocks on the full channel and stops advancing its select cursor),
		// so no DUNS is silently skipped while the subject API is unavailable.
		// We poll the breaker state so workers resume within breakerPollInterval
		// of it half-opening/closing. Previously each worker slept the full
		// CircuitBreaker.Timeout AND dropped the already-pulled task, collapsing
		// throughput to ~zero and skipping large swaths of the corpus for the
		// whole open period.
		if r.circuitBreaker.State() == gobreaker.StateOpen {
			select {
			case <-time.After(breakerPollInterval):
			case <-ctx.Done():
				return
			}
			continue
		}

		select {
		case <-ctx.Done():
			return
		case task, opened := <-input:
			if !opened {
				logger.
					Debugw("fetcher has no work left")
				return
			}
			logger := logger.With("request", task.GetRequestLink())
			logger.
				Debugw("pulling a new task", "task_count", len(input))
			activeRequests.Add(1)
			storedValues, err := r.performRequest(ctx, task, logger)
			activeRequests.Add(-1)
			if errors.Is(err, gobreaker.ErrTooManyRequests) {
				// Half-open: gobreaker admitted its MaxRequests probes and
				// rejected this call. Park until the half-open window resolves
				// instead of hot-looping — pulling and skipping tasks — while
				// the in-flight probes decide whether the breaker closes.
				for r.circuitBreaker.State() == gobreaker.StateHalfOpen {
					select {
					case <-time.After(breakerPollInterval):
					case <-ctx.Done():
						return
					}
				}
				continue
			}
			if errors.Is(err, gobreaker.ErrOpenState) {
				// Breaker opened between the top-of-loop state check and
				// execution; the next iteration's check parks us. Skip this
				// task — in continuous mode the DUNS keeps its stale corr_ts
				// and is re-selected on the next pass.
				continue
			}
			// It's expected that err is ignored here
			for _, value := range storedValues {
				output <- value
			}
			if err != nil {
				zap.S().Error(
					fmt.Errorf(
						"performing request: %w",
						err,
					),
				)
			}
		case <-time.After(r.cfg.Fetcher.IdleTime):
			logger.
				Debugw(
					"no tasks recieved in fetcher idle time, exiting fetcher",
					"idle_time",
					r.cfg.Fetcher.IdleTime,
				)
			return
		}
	}
}

func (r *Runner[S, R, P, Q]) startFetchers(
	globalWg *sync.WaitGroup,
	ctx context.Context,
	input chan APIRequest[P],
) chan S {
	outputCh := make(chan S, 2*r.cfg.Writer.InsertBatchSize+1)
	wg := sync.WaitGroup{}
	fetcherCnt := atomic.Int32{}
	for i := range r.cfg.Fetcher.MaxFetcherWorkers {
		var rnd time.Duration
		if i < r.cfg.Fetcher.MinFetcherWorkers || !r.cfg.Fetcher.EnableWarmup {
			rnd = 0
		} else {
			rnd = time.Duration(rand.IntN(int(r.cfg.Fetcher.Duration.Seconds())+1)) * time.Second
		}
		wg.Go(func() {
			<-time.After(rnd)
			fetcherCnt.Add(1)
			defer fetcherCnt.Add(-1)
			r.fetcher(ctx, input, outputCh, i)
		})
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 10):
				zap.S().
					Debugf("%d fetchers are currently running", fetcherCnt.Load())
			}
		}
	}()

	globalWg.Go(func() {
		defer close(outputCh)
		defer zap.S().Info("all fetchers have been stopped")
		wg.Wait()
	})

	return outputCh
}

func (r *Runner[S, R, P, Q]) convertToStored(
	req APIRequest[P],
	attempt AttemptData,
	attemptNumber int,
	logger *zap.SugaredLogger,
) S {
	resp := attempt.Response
	var result R
	statusCode := resp.StatusCode()
	switch statusCode {
	case 0:
		statusCode = 599
	case 429:
		// do nothing, but not default
	default:
		body := resp.Bytes()
		var tmpResult R
		err := json.Unmarshal(body, &tmpResult)
		if err != nil {
			logger.
				Warnw(
					"unmarshalling response into a response object failed, saving the status code",
					"error",
					err,
					"status_code",
					statusCode,
					"body",
					string(body[min(10, len(body)):]),
				)
		} else {
			result = tmpResult
		}
	}

	storedValue := result.IntoStored(
		req,
		attempt.Error,
		attemptNumber+1,
		statusCode,
		resp.Duration(),
		r.cfg.Writer.SaveTag,
	)

	return storedValue
}

var (
	ErrClientError = errors.New("client error from subject API")
	ErrServerError = errors.New("server error from subject API")
)

type AttemptData struct {
	Response *resty.Response
	Error    error
}

type RetryTracker struct {
	attempts []AttemptData
}

func (r *RetryTracker) Add(resp *resty.Response, err error) {
	r.attempts = append(r.attempts, AttemptData{
		Response: resp,
		Error:    err,
	})
}

func (r *RetryTracker) Attempts() []AttemptData {
	attempts := make([]AttemptData, len(r.attempts))
	copy(attempts, r.attempts)
	return attempts
}

// NOTE(nrydanov): This function is too complex, I've been thinking about it
// for a while and I'm not sure how to simplify it, sooo...
//
//gocyclo:ignore
func (r *Runner[S, R, P, Q]) performRequest(
	ctx context.Context,
	req APIRequest[P],
	logger *zap.SugaredLogger,
) ([]S, error) {
	requestURL := req.GetRequestLink()
	requestBody := req.GetRequestBody()

	processResp := func(resp *resty.Response, err error) error {
		lastStatus := resp.StatusCode()
		if lastStatus > 399 && lastStatus < 500 {
			return fmt.Errorf("%w: %v, status_code: %d", ErrClientError, resp.Error(), resp.StatusCode())
		}
		if lastStatus > 499 {
			return fmt.Errorf("%w: %v, status_code: %d", ErrServerError, resp.Error(), resp.StatusCode())
		}
		return err
	}

	var tracker RetryTracker

	// Attach the per-request retry tracker to the Request, NOT the shared client.
	// r.httpClient.AddRetryHooks mutates the shared client's retryHooks slice on
	// every request: it grows unboundedly and client.R() clones the ever-growing
	// slice on each call (slices.Clone[RetryHookFunc]) — O(N^2) memory, OOM within
	// minutes under load. Request-scoped hooks are honoured during retries too.
	request := r.httpClient.R().WithContext(ctx).AddRetryHooks(tracker.Add)
	var toBeExecuted func() (*resty.Response, error)
	switch req.Method {
	case config.RunnerHTTPMethodGet:
		toBeExecuted = func() (*resty.Response, error) {
			resp, err := request.Get(requestURL)
			return resp, processResp(resp, err)
		}
	case config.RunnerHTTPMethodPost:
		toBeExecuted = func() (*resty.Response, error) {
			resp, err := request.
				SetBody(requestBody).
				Post(requestURL)
			return resp, processResp(resp, err)
		}
	}
	lastResp, err := r.circuitBreaker.Execute(toBeExecuted)
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) ||
			errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, err
		} else {
			zap.S().Warn(fmt.Errorf("request is finished with error: %w", err))
		}
	}
	if lastResp != nil {
		tracker.Add(lastResp, err)
	} else {
		logger.Warnw("unexpected nil response after error", "error", err)
	}

	var results []S

	for i, resp := range tracker.Attempts() {
		storedValue := r.convertToStored(req, resp, i, logger)
		results = append(results, storedValue)
		logger.Debugw("response processed", "attempt", i+1)
	}
	return results, nil
}
