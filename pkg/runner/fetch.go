package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	re "orb/runner/pkg/runner/enum"
	rr "orb/runner/pkg/runner/request"

	"github.com/go-resty/resty/v2"
)

// Performs requests to the target API and returns results.
func (r *Runner[S, R, P, Q]) fetch(
	ctx context.Context,
	requests []rr.GetRequest[P],
	remainder []S,
	writerTasks chan []S,
) ([]S, [][]S) {
	logObject := log.L().Tag(log.LogTagFetching)
	input := make(chan rr.GetRequest[P], len(requests))
	output := make(chan S, config.C.Run.FetcherWorkers)

	for _, req := range requests {
		input <- req
	}
	close(input)

	workerWg := sync.WaitGroup{}
	for i := range config.C.Run.FetcherWorkers {
		workerWg.Add(1)
		go func(fetcherNum int) {
			defer workerWg.Done()
			for task := range input {
				select {
				case <-ctx.Done():
					return
				default:
					storedValues := r.handleFetcherTask(ctx, task, fetcherNum)
					for _, value := range storedValues {
						output <- value
					}
				}
			}
		}(i)
		log.S.Debug("Launched a new fetcher worker", logObject)
	}

	var processed [][]S
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for res := range output {
			remainder = append(remainder, res)
			if len(remainder) >= config.C.Run.BatchSize {
				log.S.Debug(
					"Collected enough records to write to the database",
					logObject,
				)
				processed = append(processed, remainder)
				writerTasks <- remainder
				log.S.Debug("Sent results to the writer", logObject)
				remainder = []S{}
			}
		}
	}()

	workerWg.Wait()
	close(output)
	collectorWg.Wait()

	return remainder, processed
}

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

	responses := lastResponse.
		Request.
		Context().
		Value(re.RequestContextKeyUnsuccessfulResponses).([]*resty.Response)
	if lastResponse.IsSuccess() || config.C.HttpRetries.NumRetries == 0 ||
		lastResponse.StatusCode() == 0 {
		responses = append(responses, lastResponse)
	}

	results := []S{}
	for i, response := range responses {
		var result R
		statusCode := response.StatusCode()
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
			err = json.Unmarshal(response.Body(), &result)
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
			response.Time(),
		)

		results = append(
			results,
			storedValue,
		)
	}
	return results, nil
}
