package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"orb/runner/src/api"
	"orb/runner/src/config"
	dbclient "orb/runner/src/db/clients"
	"orb/runner/src/log"
	rd "orb/runner/src/runner/data"
	re "orb/runner/src/runner/enum"
	ri "orb/runner/src/runner/interface"
	rr "orb/runner/src/runner/request"

	"github.com/avast/retry-go/v4"
	"github.com/go-resty/resty/v2"
)

type Runner[S ri.StoredValue, R ri.Response[S, P], P ri.StoredParams] struct {
	clickHouseClient     dbclient.ClickHouseClient[S, P]
	httpClient           *resty.Client
	workerTimeout        time.Duration
	qualityControlConfig config.QualityControlConfig
}

func New[S ri.StoredValue, R ri.Response[S, P], P ri.StoredParams]() *Runner[S, R, P] {
	clickHouseClient, version, err := dbclient.NewClickHouseClient[S, P](
		config.C.ClickHouse,
	)
	if err != nil {
		log.S.Errorw(
			"Connection to the ClickHouse database was unsuccessful!",
			"error", err,
			"tag", log.TagClickHouseError,
		)
		return nil
	} else {
		log.S.Infow(
			"Connection to the ClickHouse database was successful!",
			"tag", log.TagClickHouseSuccess,
		)
		log.S.Infow(
			fmt.Sprintf("%v", version),
			"tag", log.TagClickHouseSuccess,
		)
	}

	// log.S.Infow("Creating table which is required for the run")
	// TODO(evgenymng): uncomment, when actual DDL is written
	// var zeroInstance S
	// zeroInstance.GetCreateQuery()

	runner := Runner[S, R, P]{
		clickHouseClient: *clickHouseClient,
		httpClient:       initHttpClient(),
		workerTimeout: time.Duration(
			config.C.Timeouts.GoroutineTimeout,
		) * time.Second,
	}
	return &runner
}

func (r *Runner[S, R, P]) SendGetRequest(
	ctx context.Context,
	req rr.GetRequest[P],
) ([]S, error) {
	url, err := req.CreateGetRequestLink(config.C.Run.ExtraParams)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(
		ctx,
		re.RequestContextKeyUnsuccessfulResponses,
		[]*resty.Response{},
	)
	lastResponse, err := r.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
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
				return nil, err
			}
		}
		storedValue := result.IntoStored(req.Params, i+1, url, statusCode)

		results = append(
			results,
			storedValue,
		)
	}
	return results, nil
}

// Run the runner's job within a given context on a specified API.
func (r *Runner[S, R, P]) Run(ctx context.Context, service api.Api[S]) {
	// + 1 for the [nil] task
	fetcherTasks := make(
		chan *rr.GetRequest[P],
		config.C.Run.SelectionBatchSize+1,
	)
	fetcherResults := make(
		chan rd.FetcherResult[S],
		config.C.Run.SelectionBatchSize,
	)
	writtenBatches := make(
		chan rd.ProcessedBatch[S],
		config.C.Run.WriterWorkers,
	)

	nothingLeft := make(chan bool, 1)
	qcResults := make(chan rd.QualityControlResult[S], 1)
	defer close(fetcherResults)
	defer close(fetcherTasks)
	defer close(writtenBatches)
	defer close(nothingLeft)

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < config.C.Run.FetcherWorkers; i++ {
		go r.fetcher(
			workerCtx,
			i,
			fetcherTasks,
			fetcherResults,
			nothingLeft,
		)
	}
	for i := 0; i < config.C.Run.WriterWorkers; i++ {
		go r.writer(workerCtx, i, fetcherResults, writtenBatches)
	}
	go r.qualityControl(
		workerCtx,
		writtenBatches,
		qcResults,
	)

	selectedBatch := []P{}
	nothingLeft <- true

	for {
		select {
		case _, ok := <-nothingLeft:
			log.S.Debug("Got \"nothing left\" signal from one of fetchers")
			if !ok {
				return
			}

			err := retry.Do(
				func() error {
					var err error
					selectedBatch, err = r.clickHouseClient.SelectNextBatch(
						ctx,
						config.C.Run.DayOffset,
						config.C.Run.SelectionBatchSize,
					)
					return err
				},
				retry.Attempts(uint(config.C.SelectRetries.NumRetries)+1),
			)
			if err != nil {
				log.S.Errorw(
					"Failed to fetch URL parameters from the ClickHouse!",
					"error", err,
					"tag", log.TagClickHouseError,
				)
				break
			}
			log.S.Debugw(
				"Creating tasks for the fetchers",
				"tag", log.TagRunnerDebug,
			)
			for _, task := range selectedBatch[:config.C.Run.VerificationBatchSize] {
				fetcherTasks <- rr.NewGetRequest(
					config.C.Api.Host,
					config.C.Api.Port,
					config.C.Api.Method,
					task,
				)
			}
			fetcherTasks <- nil
			selectedBatch = selectedBatch[config.C.Run.VerificationBatchSize:]

		case res, ok := <-qcResults:
			if !ok {
				return
			}

			service.AfterBatch(ctx, res.Batch, &res.FailCount)

			if res.FailCount > 0 {
				log.S.Warnw(
					"Batch quality control was not passed",
					"tag", log.TagQualityControl,
					"fail_count", res.FailCount,
				)
				err := r.standby(ctx)
				if err != nil {
					return
				}
			} else {
				log.S.Infow(
					"Batch quality control has successfully been passed",
					"tag", log.TagQualityControl,
				)
			}
		}
	}
}

func (r *Runner[S, R, P]) standby(ctx context.Context) error {
	log.S.Infow("The runner has entered standby mode.")
	waitTime := time.Duration(config.C.Run.SleepTime) * time.Second
	defer log.S.Infow("The runner has left standby mode")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		return nil
	}
}

func initHttpClient() *resty.Client {
	return resty.New().SetRetryCount(config.C.HttpRetries.NumRetries).
		SetTimeout(time.Duration(time.Duration(config.C.Timeouts.VerifierTimeout) * time.Second)).
		SetRetryWaitTime(time.Duration(config.C.HttpRetries.MinWaitTime) * time.Second).
		SetRetryMaxWaitTime(time.Duration(config.C.HttpRetries.MaxWaitTime) * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				if r.StatusCode() >= http.StatusInternalServerError {
					log.S.Debugw(
						"Retrying request",
						"request_status_code", r.StatusCode(),
						"verify_url", r.Request.URL,
						"tag", log.TagErrorResponse,
					)
					return true
				}
				return false
			},
		).
		// TODO(nrydanov): Find other way to handle list of unsucessful responses
		// as using WithValue for these purposes is anti-pattern
		AddRetryHook(
			func(r *resty.Response, err error) {
				ctx := r.Request.Context()
				responses := ctx.Value(re.RequestContextKeyUnsuccessfulResponses).([]*resty.Response)
				responses = append(responses, r)
				newCtx := context.WithValue(
					ctx,
					re.RequestContextKeyUnsuccessfulResponses,
					responses,
				)
				r.Request.SetContext(newCtx)
			},
		).
		SetLogger(log.S)
}
