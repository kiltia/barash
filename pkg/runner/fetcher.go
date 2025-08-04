package runner

import (
	"context"
	"encoding/json"
	"fmt"

	"orb/runner/pkg/config"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// Performs requests to the target API and returns results.

func (r *Runner[S, R, P, Q]) handleFetcherTask(
	ctx context.Context,
	task ServiceRequest[P],
) []S {
	zap.S().
		Debugw("Sending request to the subject API", "fetcher_num", ctx.Value(ContextKeyFetcherNum))
	resultList, err := r.performRequest(ctx, task)
	if err != nil {
		zap.S().
			Errorw(
				"There was an error while sending request to the subject API",
				"error",
				err,
				"fetcher_num",
				ctx.Value(ContextKeyFetcherNum),
			)
	}
	zap.S().
		Debugw("Finished request handling", "fetcher_num", ctx.Value(ContextKeyFetcherNum))
	return resultList
}

func (r *Runner[S, R, P, Q]) sendServiceRequest(
	ctx context.Context,
	method config.RunnerHTTPMethod,
	url string,
	body map[string]any,
) *resty.Response {
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

	zap.S().Debugw("Finished request to the subject API", "url", url)
	if err != nil {
		return lastResponse
	}

	return lastResponse
}

func (r *Runner[S, R, P, Q]) processResponse(
	req ServiceRequest[P],
	resp *resty.Response,
	attemptNumber int,
) (S, error) {
	var result R
	statusCode := resp.StatusCode()
	switch statusCode {
	case 0:
		result = *new(R)
		statusCode = 599
		zap.S().
			Debugw("Timeout was reached while waiting for a response", "status_code", statusCode)
	case 429:
		result = *new(R)
		statusCode = 429
		zap.S().
			Debugw(`Subject API responded with "Too Many Requests"`, "status_code", statusCode)
	default:
		err := json.Unmarshal(resp.Body(), &result)
		if err != nil {
			zap.S().
				Warnw("Failed to unmarshal response into a response object. Only saving the status code", "error", err, "status_code", resp.StatusCode())
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
) ([]S, error) {
	requestUrl := req.GetRequestLink()
	zap.S().Debugw("Request link created", "url", requestUrl)

	requestBody := req.GetRequestBody()
	zap.S().Debugw("Request body constructed", "body", requestBody)

	lastResponse := r.sendServiceRequest(
		ctx,
		req.Method,
		requestUrl,
		requestBody,
	)
	lastStatus := lastResponse.StatusCode()

	if lastStatus >= 400 && lastStatus < 500 {
		var err error
		if lastStatus == 429 {
			err = fmt.Errorf("subject API is overloaded")
		} else {
			err = fmt.Errorf("client error from the subject API")
		}
		zap.S().
			Warnw("The subject API responded with 4xx. You should probably check your configuration", "status_code", lastStatus, "error", err, "response", lastResponse.String())
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
		zap.S().Debugw("Processing response", "attempt", i)
		storedValue, err := r.processResponse(req, resp, i)
		if err != nil {
			return nil, err
		}
		results = append(results, storedValue)
		zap.S().Debugw("Response processed", "attempt", i)
	}
	return results, nil
}

func (r *Runner[S, R, P, Q]) fetcher(
	ctx context.Context,
	input chan ServiceRequest[P],
	output chan S,
	fetcherNum int,
) {
	zap.S().
		Debugw("A new fetcher instance is starting up", "fetcher_num", fetcherNum)
	ctx = context.WithValue(ctx, ContextKeyFetcherNum, fetcherNum)

	for {
		select {
		case task, opened := <-input:
			if !opened {
				zap.S().
					Infow("Fetcher has no work left", "fetcher_num", fetcherNum)
				return
			}
			zap.S().
				Debugw("Pulling a new task", "task_count", len(input), "fetcher_num", fetcherNum)
			storedValues := r.handleFetcherTask(ctx, task)
			for _, value := range storedValues {
				zap.S().
					Debugw("Sending fetch result to writer", "fetcher_num", fetcherNum)
				output <- value
				zap.S().
					Debugw("Result to writer was sent", "fetcher_num", fetcherNum)
			}
			zap.S().
				Debugw("Finished sending results to writer", "fetcher_num", fetcherNum)
		case <-ctx.Done():
			return
		}
	}
}
