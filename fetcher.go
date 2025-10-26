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

func (r *Runner[S, R, P, Q]) fetcher(
	ctx context.Context,
	input <-chan APIRequest[P],
	output chan<- S,
	fetcherNum int,
) {
	logger := zap.S().
		With("fetcher_num", fetcherNum)
	logger.
		Debugw("fetcher instance is starting up")

	ctx = context.WithValue(ctx, ContextKeyFetcherNum, fetcherNum)

	activeRequests := atomic.Int32{}
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 10):
			zap.S().
				Debugf("%d active requests are currently running", activeRequests.Load())
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			select {
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
				if errors.Is(err, gobreaker.ErrOpenState) ||
					errors.Is(err, gobreaker.ErrTooManyRequests) {
					zap.S().
						Warnw("fetcher is paused after too many client/server errors")
					select {
					case <-time.After(r.cfg.Fetcher.CircuitBreaker.Timeout):
					case <-ctx.Done():
						return
					}
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
			return fmt.Errorf("%w: %v", ErrClientError, resp.Error())
		}
		if lastStatus > 499 {
			return fmt.Errorf("%w: %v", ErrServerError, resp.Error())
		}
		return err
	}

	var tracker RetryTracker

	client := r.httpClient.AddRetryHooks(tracker.Add)

	request := client.R().WithContext(ctx)
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
