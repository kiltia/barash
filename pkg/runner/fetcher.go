package runner

import (
	"context"
	"encoding/json"
	"fmt"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"

	"github.com/go-resty/resty/v2"
)

// Performs requests to the target API and returns results.

func (r *Runner[S, R, P, Q]) handleFetcherTask(
	ctx context.Context,
	logObject log.LogObject,
	task GetRequest[P],
) []S {
	log.S.Debug("Sending request to the subject API", logObject)
	resultList, err := r.performRequest(ctx, logObject, task)
	if err != nil {
		log.S.Error(
			"There was an error while sending request to the subject API",
			logObject.Error(err),
		)
	}
	log.S.Debug("Finished request handling", logObject)
	return resultList
}

func (r *Runner[S, R, P, Q]) sendGetRequest(
	ctx context.Context,
	logObject log.LogObject,
	url string,
) *resty.Response {
	log.S.Debug("Performing request to the subject API", logObject)
	ctx = context.WithValue(
		ctx,
		ContextKeyUnsuccessfulResponses,
		[]*resty.Response{},
	)
	lastResponse, err := r.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return lastResponse
	}
	log.S.Debug("Finished request to the subject API", logObject)

	return lastResponse
}

func (r *Runner[S, R, P, Q]) processResponse(
	logObject log.LogObject,
	req GetRequest[P],
	resp *resty.Response,
	attemptNumber int,
) (S, error) {
	var result R
	statusCode := resp.StatusCode()
	url := req.Params.GetUrl()
	if statusCode == 0 {
		result = *new(R)
		statusCode = 599
		log.S.Debug(
			"Timeout was reached while waiting for a response",
			logObject.
				Error(fmt.Errorf("timeout reached")).
				Add("url", url),
		)
	} else if statusCode == 429 {
		result = *new(R)
		statusCode = 429
		log.S.Debug(
			`Subject API responded with "Too Many Requests"`,
			logObject.
				Error(fmt.Errorf("too many requests")).
				Add("url", url),
		)
	} else {
		err := json.Unmarshal(resp.Body(), &result)
		if err != nil {
			log.S.Warn(
				"Failed to unmarshal response into a response object. "+
					"Only saving the status code.",
				logObject.Error(err),
			)
			result = *new(R)
			statusCode = resp.StatusCode()
		}
	}

	requestLink, err := req.GetRequestLink()
	if err != nil {
		return *new(S), err
	}

	storedValue := result.IntoStored(
		req.Params,
		attemptNumber+1,
		requestLink,
		statusCode,
		resp.Time(),
	)

	return storedValue, nil
}

// NOTE(nrydanov): This function is too complex, I've been thinking about it
// for a while and I'm not sure how to simplify it, sooo...
//
//gocyclo:ignore
func (r *Runner[S, R, P, Q]) performRequest(
	ctx context.Context,
	logObject log.LogObject,
	req GetRequest[P],
) ([]S, error) {
	log.S.Debug("Creating request link", logObject)
	requestUrl, err := req.GetRequestLink()
	if err != nil {
		log.S.Error("Failed to create request link", logObject.Error(err))
		return nil, err
	}
	log.S.Debug(
		"Request link successfully created",
		logObject.Add("url", requestUrl),
	)

	lastResponse := r.sendGetRequest(ctx, logObject, requestUrl)
	lastStatus := lastResponse.StatusCode()

	if lastStatus >= 400 && lastStatus < 500 {
		var err error
		if lastStatus == 429 {
			err = fmt.Errorf("subject API is overloaded")
		} else {
			err = fmt.Errorf("client error from the subject API")
		}
		log.S.Warn(
			"The subject API responded with 4xx. "+
				"You should probably check your configuration.",
			logObject.Add("status_code", lastStatus).Error(err),
		)
	}

	var responses []*resty.Response
	// NOTE(evgenymng): sorry, I know this is garbage
	if lastResponse.IsSuccess() || config.C.HttpRetries.NumRetries == 0 ||
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
		log.S.Debug("Processing response...", logObject)
		storedValue, err := r.processResponse(logObject, req, resp, i)
		if err != nil {
			return nil, err
		}
		results = append(results, storedValue)
		log.S.Debug("Response processed", logObject)
	}
	return results, nil
}

func (r *Runner[S, R, P, Q]) fetcher(
	ctx context.Context,
	input chan GetRequest[P],
	output chan S,
	fetcherNum int,
) {
	logObject := log.L().Tag(log.LogTagFetching).Add("fetcher_num", fetcherNum)
	log.S.Debug("A new fetcher instance is starting up", logObject)
	ctx = context.WithValue(ctx, ContextKeyFetcherNum, fetcherNum)

	for {
		select {
		case task, opened := <-input:
			if !opened {
				log.S.Info("Fetcher has no work left", logObject)
				return
			}
			log.S.Debug(
				"Pulling a new task",
				logObject.Add("task_count", len(input)),
			)
			storedValues := r.handleFetcherTask(ctx, logObject, task)
			for _, value := range storedValues {
				log.S.Debug("Sending fetch result to writer", logObject)
				output <- value
				log.S.Debug("Result to writer was sent", logObject)
			}
			log.S.Debug("Finished sending results to writer", logObject)
		case <-ctx.Done():
			return
		}
	}
}
