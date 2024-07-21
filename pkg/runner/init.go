package runner

import (
	"context"
	"net/http"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	re "orb/runner/pkg/runner/enum"

	"github.com/go-resty/resty/v2"
)

func (r *Runner[S, R, P, Q]) initTable(ctx context.Context) {
	if config.C.Run.Mode == config.ContinuousMode {
		log.S.Info("Running in continuous mode, skipping table initialization")
		return
	}
	var nilInstance S
	err := r.clickHouseClient.Connection.Exec(ctx, nilInstance.GetCreateQuery())

	if err != nil {
		log.S.Warnw("Table creation script has failed", "reason", err)
	} else {
		log.S.Info("Successfully initialized table for the Runner results")
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
