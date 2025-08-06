package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kiltia/runner/pkg/config"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// Performs requests to the target API and returns results.

func (r *Runner[S, R, P, Q]) sendServiceRequest(
	ctx context.Context,
	method config.RunnerHTTPMethod,
	url string,
	body map[string]any,
) (*resty.Response, error) {
	zap.S().Debugw("Performing request to the subject API", "url", url)
	ctx = context.WithValue(
		ctx,
		ContextKeyUnsuccessfulResponses,
		[]*resty.Response{},
	)

	var lastResponse *resty.Response
	var err error

	switch method {
	case config.RunnerHTTPMethodGet:
		lastResponse, err = r.httpClient.R().SetContext(ctx).Get(url)
	case config.RunnerHTTPMethodPost:
		lastResponse, err = r.httpClient.R().SetContext(ctx).
			SetBody(body).
			Post(url)
	}

	if err != nil {
		return lastResponse, err
	}

	return lastResponse, err
}

func (r *Runner[S, R, P, Q]) processResponse(
	req ServiceRequest[P],
	resp *resty.Response,
	attemptNumber int,
	logger *zap.SugaredLogger,
) (S, error) {
	var result R
	statusCode := resp.StatusCode()
	switch statusCode {
	case 0:
		result = *new(R)
		statusCode = 599
	case 429:
		result = *new(R)
		statusCode = 429
	default:
		body := resp.Body()
		err := json.Unmarshal(body, &result)
		if err != nil {
			logger.
				Warnw(
					"unmarshalling response into a response object failed, saving the status code",
					"error",
					err,
					"status_code",
					resp.StatusCode(),
					"body",
					body,
				)
			result = *new(R)
			statusCode = resp.StatusCode()
		}
	}

	requestLink := req.GetRequestLink()
	requestBody := req.GetRequestBody()

	storedValue := result.IntoStored(
		req.Params,
		attemptNumber+1,
		requestLink,
		requestBody,
		statusCode,
		resp.Time(),
		r.cfg.Run.Tag,
	)

	return storedValue, nil
}

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

	lastResponse, err := r.sendServiceRequest(
		ctx,
		req.Method,
		requestURL,
		requestBody,
	)
	lastStatus := lastResponse.StatusCode()

	if lastStatus >= 400 && lastStatus < 500 {
		if lastStatus == 429 {
			err = fmt.Errorf("subject API is overloaded")
		} else {
			err = fmt.Errorf("client error from the subject API: %v", err)
		}
	}

	var responses []*resty.Response
	// NOTE(evgenymng): sorry, I know this is garbage
	if lastResponse.IsSuccess() || r.cfg.HTTPRetries.NumRetries == 0 ||
		lastResponse.StatusCode() == 0 || lastResponse.StatusCode() == 429 {
		responses = append(responses, lastResponse)
	}

	failed := lastResponse.
		Request.
		Context().
		Value(ContextKeyUnsuccessfulResponses).([]*resty.Response)
	responses = append(responses, failed...)

	var results []S

	for i, resp := range responses {
		zap.S().Debugw("processing response", "attempt", i)
		storedValue, err := r.processResponse(req, resp, i, logger)
		if err != nil {
			return nil, err
		}
		results = append(results, storedValue)
		zap.S().Debugw("response processed", "attempt", i)
	}
	return results, err
}

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

	innerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	innerCtx = context.WithValue(innerCtx, ContextKeyFetcherNum, fetcherNum)

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
				logger.
					Debugw("pulling a new task", "task_count", len(input))
				storedValues, err := r.performRequest(innerCtx, task, logger)
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
			case <-time.After(r.cfg.Run.FetcherIdleTime):
				logger.
					Debugw(
						"no tasks recieved in fetcher idle time, exiting fetcher",
						"idle_time",
						r.cfg.Run.FetcherIdleTime,
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
	outputCh := make(chan S, 2*r.cfg.Run.InsertionBatchSize+1)
	wg := sync.WaitGroup{}
	wg.Add(r.cfg.Run.MaxFetcherWorkers)
	fetcherCnt := atomic.Int32{}
	for i := range r.cfg.Run.MaxFetcherWorkers {
		var rnd time.Duration
		if i < r.cfg.Run.MinFetcherWorkers {
			rnd = 0
		} else {
			rnd = time.Duration(rand.IntN(int(r.cfg.Run.WarmupTime.Seconds())+1)) * time.Second
		}
		go func() {
			defer wg.Done()
			<-time.After(rnd)
			fetcherCnt.Add(1)
			r.fetcher(ctx, input, outputCh, i)
		}()
	}

	go func() {
		for {
			time.Sleep(time.Second * 10)
			zap.S().Infow(
				"current fetcher count",
				"fetcher_cnt", fetcherCnt.Load(),
			)
		}
	}()

	go func() {
		defer globalWg.Done()
		wg.Wait()
	}()

	return outputCh
}
