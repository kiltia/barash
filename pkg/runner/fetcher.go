package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	re "orb/runner/pkg/runner/enum"
	rr "orb/runner/pkg/runner/request"

	"github.com/go-resty/resty/v2"
)

// Performs requests to the target API and returns results.

func (r *Runner[S, R, P, Q]) handleFetcherTask(
	ctx context.Context,
	task rr.GetRequest[P],
	fetcherNum int,
) []S {
	logObject := log.L().Tag(log.LogTagFetching).Add("fetcher_num", fetcherNum)

	log.S.Debug("Sending request to the subject API", logObject)
	resultList, err := r.sendGetRequest(ctx, task, fetcherNum)
	if err != nil {
		log.S.Error(
			"There was an error while sending request to the subject API",
			logObject.Error(err),
		)
	}
	return resultList
}

func (r *Runner[S, R, P, Q]) sendGetRequest(
	ctx context.Context,
	req rr.GetRequest[P],
	fetcherNum int,
) ([]S, error) {
	logObject := log.L().Tag(log.LogTagFetching).Add("fetcher_num", fetcherNum)

	log.S.Debug("Creating request link", logObject)
	url, err := req.CreateGetRequestLink(config.C.Run.ExtraParams)
	if err != nil {
		log.S.Error("Failed to create request link", logObject.Error(err))
		return nil, err
	}
	log.S.Debug("Request link successfully created", logObject.Add("url", url))

	log.S.Debug("Performing request to the subject API", logObject)
	ctx = context.WithValue(
		ctx,
		re.RequestContextKeyUnsuccessfulResponses,
		[]*resty.Response{},
	)
	ctx = context.WithValue(
		ctx,
		re.RequestContextKeyFetcherNum,
		fetcherNum,
	)
	lastResponse, err := r.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		log.S.Error("Failed to perform the request", logObject.Error(err))
	}
	log.S.Debug("Finished request to the subject API", logObject)

	var responses []*resty.Response
	lastStatus := lastResponse.StatusCode()
	if lastStatus >= 400 && lastStatus < 500 {
		err := fmt.Errorf("client error from the subject API")
		log.S.Error(
			"The subject API responded with 4xx. "+
				"You should probably check your configuration.",
			logObject.Add("status_code", lastStatus).
				Error(err),
		)
		return nil, err
	}

	if lastResponse.IsSuccess() || config.C.HttpRetries.NumRetries == 0 ||
		lastResponse.StatusCode() == 0 {
		responses = append(responses, lastResponse)
	}

	failed := lastResponse.
		Request.
		Context().
		Value(re.RequestContextKeyUnsuccessfulResponses).([]*resty.Response)
	responses = append(responses, failed...)

	var results []S
	for i, resp := range responses {
		var result R
		statusCode := resp.StatusCode()
		if statusCode == 0 {
			result = *new(R)
			statusCode = 599
			log.S.Debug(
				"Timeout was reached while waiting for a response",
				logObject.
					Error(fmt.Errorf("timeout reached")).
					Add("url", url),
			)
		} else {
			err = json.Unmarshal(resp.Body(), &result)
			if err != nil {
				log.S.Error(
					"Failed to unmarshal the response",
					logObject.Error(err),
				)
				return nil, err
			}
		}
		storedValue := result.IntoStored(
			req.Params,
			i+1,
			url,
			statusCode,
			resp.Time(),
		)

		results = append(results, storedValue)
	}
	return results, nil
}

func (r *Runner[S, R, P, Q]) fetcher(
	ctx context.Context,
	input chan rr.GetRequest[P],
	output chan S,
	standbyChannel chan bool,
	fetcherNum int,
) {
	logObject := log.L().Tag(log.LogTagFetching).Add("fetcher_num", fetcherNum)
	for {
		select {
		case <-standbyChannel:
			log.S.Debug("Fetcher got standby signal, sleeping...", logObject)
			err := r.standby(ctx)
			log.S.Debug("Fetcher left the standby mode", logObject)
			if err != nil {
				return
			}
		default:
			select {
			case task := <-input:
                log.S.Debug("Pulling a new task", logObject.Add("task_count", len(input)))
				storedValues := r.handleFetcherTask(ctx, task, fetcherNum)
				for _, value := range storedValues {
					output <- value
				}
			case <-ctx.Done():
				return
			default:
				log.S.Info("Got nothing to fetch, time to sleep", logObject)
				// TODO(nrydanov): Replace with config value
				time.Sleep(5 * time.Second)
			}
		}
	}
}
