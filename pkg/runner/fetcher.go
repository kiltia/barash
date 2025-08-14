package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kiltia/runner/pkg/config"
	"github.com/sony/gobreaker/v2"

	"go.uber.org/zap"
	"resty.dev/v3"
)

func (r *Runner[S, R, P, Q]) fetcher(
	ctx context.Context,
	input chan ServiceRequest[P],
	output chan S,
	fetcherNum int,
) {
	logger := zap.S().
		With("fetcher_num", fetcherNum)
	logger.
		Debugw("fetcher instance is starting up")

	ctx = context.WithValue(ctx, ContextKeyFetcherNum, fetcherNum)

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
				storedValues, err := r.performRequest(ctx, task, logger)
				if errors.Is(err, gobreaker.ErrOpenState) {
					zap.S().
						Warnw("fetcher is paused after too many client/server errors")
					select {
					case <-time.After(r.cfg.CircuitBreaker.Timeout):
					case <-ctx.Done():
						return
					}
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
	ctx context.Context,
	input chan ServiceRequest[P],
	globalWg *sync.WaitGroup,
) chan S {
	outputCh := make(chan S, 2*r.cfg.Writer.InsertionBatchSize+1)
	wg := sync.WaitGroup{}
	wg.Add(r.cfg.Fetcher.MaxFetcherWorkers)
	fetcherCnt := atomic.Int32{}
	for i := range r.cfg.Fetcher.MaxFetcherWorkers {
		var rnd time.Duration
		if i < r.cfg.Fetcher.MinFetcherWorkers || !r.cfg.Fetcher.EnableWarmup {
			rnd = 0
		} else {
			rnd = time.Duration(rand.IntN(int(r.cfg.Fetcher.Duration.Seconds())+1)) * time.Second
		}
		go func() {
			defer wg.Done()
			<-time.After(rnd)
			fetcherCnt.Add(1)
			defer fetcherCnt.Add(-1)
			r.fetcher(ctx, input, outputCh, i)
		}()
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

	go func() {
		defer globalWg.Done()
		defer close(outputCh)
		defer zap.S().Info("all fetchers have been stopped")
		wg.Wait()
	}()

	return outputCh
}

func (r *Runner[S, R, P, Q]) convertToStored(
	req ServiceRequest[P],
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
					string(body[min(100, len(body)):]),
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
		r.cfg.Writer.InsertTag,
	)

	return storedValue
}

var (
	ErrClientError = errors.New("client error from subject API")
	ErrServerError = errors.New("server error from subject API")
)

// NOTE(nrydanov): This function is too complex, I've been thinking about it
// for a while and I'm not sure how to simplify it, sooo...
//
//gocyclo:ignore
func (r *Runner[S, R, P, Q]) performRequest(
	ctx context.Context,
	req ServiceRequest[P],
	logger *zap.SugaredLogger,
) ([]S, error) {
	requestURL := req.GetRequestLink()
	requestBody := req.GetRequestBody()

	processResp := func(resp *resty.Response, err error) error {
		lastStatus := resp.StatusCode()
		if lastStatus > 399 && lastStatus < 500 {
			return fmt.Errorf("%w: %v", ErrClientError, err)
		}
		if lastStatus > 499 {
			return fmt.Errorf("%w: %v", ErrServerError, err)
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
		zap.S().Warn(fmt.Printf("request is finished with error: %v", err))
		if errors.Is(err, gobreaker.ErrOpenState) {
			return nil, err
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
