package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"orb/runner/src/config"
	dbclient "orb/runner/src/db/clients"
	"orb/runner/src/log"
	rd "orb/runner/src/runner/data"
	re "orb/runner/src/runner/enum"
	"orb/runner/src/runner/hooks"
	ri "orb/runner/src/runner/interface"
	rr "orb/runner/src/runner/request"

	"github.com/avast/retry-go/v4"
	"github.com/go-resty/resty/v2"
)

type Runner[S ri.StoredValue, R ri.Response[S, P], P ri.StoredParams] struct {
	clickHouseClient dbclient.ClickHouseClient[S, P]
	httpClient       *resty.Client
	workerTimeout    time.Duration
	hooks            hooks.Hooks[S]
}

func New[
	S ri.StoredValue,
	R ri.Response[S, P],
	P ri.StoredParams,
](hs hooks.Hooks[S]) (*Runner[S, R, P], error) {
	clickHouseClient, version, err := dbclient.NewClickHouseClient[S, P](
		config.C.ClickHouse,
	)
	if err != nil {
		log.S.Errorw(
			"Connection to the ClickHouse database was unsuccessful!",
			"error", err,
			"tag", log.TagClickHouseError,
		)
		return nil, err
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

	runner := Runner[S, R, P]{
		clickHouseClient: *clickHouseClient,
		httpClient:       initHttpClient(),
		workerTimeout: time.Duration(
			config.C.Timeouts.GoroutineTimeout,
		) * time.Second,
		hooks: hs,
	}
	return &runner, nil
}

func (r *Runner[S, R, P]) SendGetRequest(
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
		log.S.Error("Got an error while completing request", "error", err)
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

// Run the runner's job within a given context.
func (r *Runner[S, R, P]) Run(ctx context.Context) {
	// + 1 for the [nil] task
	fetcherTasks := make(
		chan *rr.GetRequest[P],
		config.C.Run.RequestBatchSize+1,
	)
	fetcherResults := make(
		chan rd.FetcherResult[S],
		config.C.Run.RequestBatchSize,
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

	r.initTable(ctx)

	var inProgress sync.Map
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
		go r.writer(workerCtx, i, fetcherResults, writtenBatches, &inProgress)
	}
	go r.qualityControl(
		workerCtx,
		writtenBatches,
		qcResults,
	)

	nothingLeft <- true
	batchCounter := 0

	for {
		select {
		case _, ok := <-nothingLeft:
			log.S.Debug(`Got "nothing left" signal from one of fetchers`)
			if !ok {
				return
			}

			var selectedBatch []P
			err := retry.Do(
				func() (err error) {
					selectedBatch, err = r.clickHouseClient.SelectNextBatch(
						ctx,
						batchCounter,
					)

					filteredBatch := *new([]P)
					for _, rd := range selectedBatch {
						stored := lockUrl(rd.GetUrl(), &inProgress)
						if stored {
							filteredBatch = append(filteredBatch, rd)
						} else {
							log.S.Debugw("Batch contains URL that is being processing, filtering out.", "url", rd.GetUrl())
						}
					}
					selectedBatch = filteredBatch
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
			batchCounter += 1
			log.S.Debugw(
				"Creating tasks for the fetchers",
				"tag", log.TagRunnerDebug,
			)
			for _, task := range selectedBatch {
				fetcherTasks <- rr.NewGetRequest(
					config.C.Api.Host,
					config.C.Api.Port,
					config.C.Api.Method,
					task,
				)
			}
			fetcherTasks <- nil

		case res, ok := <-qcResults:
			if !ok {
				return
			}

			r.hooks.AfterBatch(ctx, res.Batch, &res.FailCount)

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
	log.S.Infow("The runner is entering standby mode")
	waitTime := time.Duration(config.C.Run.SleepTime) * time.Second
	defer log.S.Infow("The runner has left standby mode")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		return nil
	}
}

func (r *Runner[S, R, P]) initTable(ctx context.Context) {
	if config.C.Api.Mode == config.ContiniousMode {
		log.S.Info("Running in continious mode, skipping initialization")
		return
	}
	var nilInstance S
	err := r.clickHouseClient.Connection.Exec(ctx, nilInstance.GetCreateQuery())

	if err != nil {
		log.S.Warnw("Table creation script has failed", "reason", err)
	} else {
		log.S.Info("Successfully initialized table for Runner results")
	}
}

func initHttpClient() *resty.Client {
	return resty.New().SetRetryCount(config.C.HttpRetries.NumRetries).
		SetTimeout(time.Duration(time.Duration(config.C.Timeouts.ApiTimeout) * time.Second)).
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
		// as using WithValue for these purposes seems like anti-pattern
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
