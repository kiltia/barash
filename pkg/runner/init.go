package runner

import (
	"context"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	re "orb/runner/pkg/runner/enum"

	"github.com/go-resty/resty/v2"
)

func (r *Runner[S, R, P, Q]) initTable(ctx context.Context) {
	logObject := log.L().Tag(log.LogTagRunner)
	if config.C.Run.Mode == config.ContinuousMode {
		log.S.Info(
			"Running in continuous mode, skipping table initialization",
			logObject,
		)
		return
	}
	var nilInstance S
	err := r.clickHouseClient.Connection.Exec(ctx, nilInstance.GetCreateQuery())

	if err != nil {
		log.S.Warn(
			"Table creation script has failed",
			logObject.Error(err),
		)
	} else {
		log.S.Info(
			"Successfully initialized table for the Runner results",
			logObject,
		)
	}
}

func initHttpClient() *resty.Client {
	return resty.New().SetRetryCount(config.C.HttpRetries.NumRetries).
		SetTimeout(time.Duration(time.Duration(config.C.Timeouts.ApiTimeout) * time.Second)).
		SetRetryWaitTime(time.Duration(config.C.HttpRetries.MinWaitTime) * time.Second).
		SetRetryMaxWaitTime(time.Duration(config.C.HttpRetries.MaxWaitTime) * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				ctx := r.Request.Context()
				fetcherNum := ctx.Value(re.RequestContextKeyFetcherNum).(int)
				if r.StatusCode() >= 500 {
					log.S.Debug(
						"Retrying request",
						log.L().Tag(log.LogTagFetching).
							Add("fetcher_num", fetcherNum).
							Add("request_status_code", r.StatusCode()).
							Add("url", r.Request.URL),
					)
					return true
				}
				return false
			},
		).
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
		SetLogger(log.S.GetInternal())
}
