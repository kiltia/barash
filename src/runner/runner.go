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
	"go.uber.org/zap"
)

type Runner[S ri.StoredValueType, R ri.ResponseType[S, P], P ri.ParamsType] struct {
	clickHouseClient     dbclient.ClickHouseClient[S, P]
	apiConfig            config.ApiConfig
	httpClient           *resty.Client
	workerTimeout        time.Duration
	httpRetries          config.RetryConfig
	runConfig            config.RunConfig
	selectRetries        config.RetryConfig
	logger               *zap.SugaredLogger
	qualityControlConfig config.QualityControlConfig
}

func NewRunner[S ri.StoredValueType, R ri.ResponseType[S, P], P ri.ParamsType](
	config config.RunnerConfig,
) *Runner[S, R, P] {
	logger := zap.Must(config.LoggerConfig.Build()).Sugar()

	clickHouseClient, version, err := dbclient.NewClickHouseClient[S, P](
		config.ClickHouseConfig,
	)
	if err != nil {
		logger.Errorw(
			"Connection to the ClickHouse database was unsuccessful!",
			"error", err,
			"tag", log.TagClickHouseError,
		)
		return nil
	} else {
		logger.Infow(
			"Connection to the ClickHouse database was successful!",
			"tag", log.TagClickHouseSuccess,
		)
		logger.Infow(
			fmt.Sprintf("%v", version),
			"tag", log.TagClickHouseSuccess,
		)
	}

	// logger.Infow("Creating table which is required for the run")
	// TODO(evgenymng): uncomment, when actual DDL is written
	// var zeroInstance S
	// zeroInstance.GetCreateQuery()

	httpClient := initHttpClient(config, logger)

	runner := Runner[S, R, P]{
		clickHouseClient: *clickHouseClient,
		apiConfig:        config.ApiConfig,
		httpClient:       httpClient,
		workerTimeout: time.Duration(
			config.Timeouts.GoroutineTimeout,
		) * time.Second,
		runConfig:            config.RunConfig,
		httpRetries:          config.HttpRetries,
		selectRetries:        config.SelectRetries,
		qualityControlConfig: config.QualityControlConfig,
		logger:               logger,
	}
	return &runner
}

func (r *Runner[S, R, P]) SendGetRequest(
	ctx context.Context,
	req rr.GetRequest[P],
) ([]S, error) {
	url, err := req.CreateGetRequestLink(r.runConfig.ExtraParams)
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
	if lastResponse.IsSuccess() || r.httpRetries.NumRetries == 0 ||
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
			r.logger.Debugw(
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
		storedValue := result.IntoWith(req.Params, i+1, url, statusCode)

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
		r.runConfig.SelectionBatchSize+1,
	)
	fetcherResults := make(
		chan rd.FetcherResult[S],
		r.runConfig.SelectionBatchSize,
	)
	writtenBatches := make(
		chan rd.ProcessedBatch[S],
		r.runConfig.WriterWorkers,
	)

	nothingLeft := make(chan bool, 1)
	qcResults := make(chan rd.QualityControlResult[S], 1)
	defer close(fetcherResults)
	defer close(fetcherTasks)
	defer close(writtenBatches)
	defer close(nothingLeft)

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < r.runConfig.FetcherWorkers; i++ {
		go r.fetcher(
			workerCtx,
			i,
			fetcherTasks,
			fetcherResults,
			nothingLeft,
		)
	}
	for i := 0; i < r.runConfig.WriterWorkers; i++ {
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
			r.logger.Debug("Got \"nothing left\" signal from one of fetchers")
			if !ok {
				// TODO(nrydanov): What should we do when channels are closed?
				return
			}

			err := retry.Do(
				func() error {
					var err error
					selectedBatch, err = r.clickHouseClient.SelectNextBatch(
						ctx,
						r.runConfig.DayOffset,
						r.runConfig.SelectionBatchSize,
					)
					return err
				},
				retry.Attempts(uint(r.selectRetries.NumRetries)+1),
			)
			if err != nil {
				r.logger.Errorw(
					"Failed to fetch URL parameters from the ClickHouse!",
					"error", err,
					"tag", log.TagClickHouseError,
				)
				break
			}
			r.logger.Debugw(
				"Creating tasks for the fetchers",
				"tag", log.TagRunnerDebug,
			)
			for _, task := range selectedBatch[:r.runConfig.VerificationBatchSize] {
				fetcherTasks <- rr.NewGetRequest(
					r.apiConfig.Host,
					r.apiConfig.Port,
					r.apiConfig.Method,
					task,
				)
			}
			fetcherTasks <- nil
			selectedBatch = selectedBatch[r.runConfig.VerificationBatchSize:]

		case res, ok := <-qcResults:
			if !ok {
				return
			}

			service.AfterBatch(ctx, res.Batch)

			if res.FailCount > 0 {
				r.logger.Warnw(
					"Batch quality control was not passed",
					"tag", log.TagQualityControl,
					"fail_count", res.FailCount,
				)
				err := r.standby(ctx)
				if err != nil {
					return
				}
			} else {
				r.logger.Infow(
					"Batch quality control has successfully been passed",
					"tag", log.TagQualityControl,
				)
			}
		}
	}
}

func (r *Runner[S, R, P]) standby(ctx context.Context) error {
	r.logger.Infow("The runner has entered standby mode.")
	waitTime := time.Duration(r.runConfig.SleepTime) * time.Second
	defer r.logger.Infow("The runner has left standby mode")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		return nil
	}
}

func initHttpClient(
	config config.RunnerConfig,
	logger *zap.SugaredLogger,
) *resty.Client {
	return resty.New().SetRetryCount(config.HttpRetries.NumRetries).
		SetTimeout(time.Duration(time.Duration(config.Timeouts.VerifierTimeout) * time.Second)).
		SetRetryWaitTime(time.Duration(config.HttpRetries.MinWaitTime) * time.Second).
		SetRetryMaxWaitTime(time.Duration(config.HttpRetries.MaxWaitTime) * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				if r.StatusCode() >= http.StatusInternalServerError {
					logger.Debugw(
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
		SetLogger(logger)
}
