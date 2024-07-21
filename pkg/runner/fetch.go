package runner

import (
	"context"
	"encoding/json"
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
	}

	var processed [][]S
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for res := range output {
			remainder = append(remainder, res)
			if len(remainder) >= config.C.Run.BatchSize {
				processed = append(processed, remainder)
				writerTasks <- remainder
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
	log.S.Debugw(
		"Sending request to get page contents",
		"fetcher_num", fetcherNum,
	)
	resultList, err := r.sendGetRequest(ctx, task)
	if err != nil {
		log.S.Errorw(
			"There was an error, while sending the request "+
				"to the subject API",
			"error", err,
			"fetcher_num", fetcherNum,
		)
	}
	return resultList
}

func (r *Runner[S, R, P, Q]) sendGetRequest(
	ctx context.Context,
	req rr.GetRequest[P],
) ([]S, error) {
	url, err := req.CreateGetRequestLink(config.C.Run.ExtraParams)
	if err != nil {
		log.S.Error("Got an error while creating a link", "error", err)
		return nil, err
	}

	ctx = context.WithValue(
		ctx,
		re.RequestContextKeyUnsuccessfulResponses,
		[]*resty.Response{},
	)
	lastResponse, err := r.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		log.S.Errorw("Got an error while completing request", "error", err)
	}

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
			log.S.Debugw(
				"Timeout was reached while waiting for a request",
				"url", url,
				"error", "TIMEOUT REACHED",
				"tag", log.TagResponseTimeout,
			)
		} else {
			err = json.Unmarshal(response.Body(), &result)
			if err != nil {
				log.S.Error("Got an error while unmarshalling response", "error", err)
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
