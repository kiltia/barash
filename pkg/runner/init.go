package runner

import (
	"context"

	"github.com/kiltia/runner/pkg/config"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

func (r *Runner[S, R, P, Q]) initTable(
	ctx context.Context,
) {
	if r.cfg.Run.Mode == config.ContinuousMode {
		zap.S().
			Infow("Running in continuous mode, skipping table initialization")
		return
	}
	var nilInstance S
	err := r.clickHouseClient.Connection.Exec(
		ctx,
		nilInstance.GetCreateQuery(r.cfg.Run.InsertionTableName),
	)

	if err != nil {
		zap.S().Warnw("Table creation script has failed", "error", err)
	} else {
		zap.S().Infow("Successfully initialized table for the Runner results")
	}
}

func initHTTPClient(
	httpRetries config.HTTPRetryConfig,
	timeouts config.TimeoutConfig,
) *resty.Client {
	return resty.New().
		SetRetryCount(httpRetries.NumRetries).
		SetTimeout(timeouts.APITimeout).
		SetRetryWaitTime(httpRetries.MinWaitTime).
		SetRetryMaxWaitTime(httpRetries.MaxWaitTime).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				ctx := r.Request.Context()
				fetcherNum := ctx.Value(ContextKeyFetcherNum).(int)
				if r.StatusCode() >= 500 {
					zap.S().
						Debugw("Retrying request", "fetcher_num", fetcherNum, "status_code", r.StatusCode(), "url", r.Request.URL)
					return true
				}
				return false
			},
		).
		AddRetryHook(
			func(r *resty.Response, err error) {
				ctx := r.Request.Context()
				responses := ctx.Value(ContextKeyUnsuccessfulResponses).([]*resty.Response)
				responses = append(
					responses,
					r,
				)
				newCtx := context.WithValue(
					ctx,
					ContextKeyUnsuccessfulResponses,
					responses,
				)
				r.Request.SetContext(
					newCtx,
				)
			},
		).
		SetLogger(zap.S())
}
