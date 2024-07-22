package runner

import (
	"context"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	rd "orb/runner/pkg/runner/data"
	"orb/runner/pkg/util"
)

// Performs QC for each batch and goes into standby, if
// there were any fails.
func (r *Runner[S, R, P, Q]) qualityControl(
	ctx context.Context,
	processed [][]S,
	timestamp time.Time,
) error {
	logObject := log.L().Tag(log.LogTagQualityControl)
	totalFails := 0
	for _, batch := range processed {
		report := r.qcReport(batch, time.Since(timestamp))

		// call user-defined logic (if any)
		r.hooks.AfterBatch(ctx, batch, &report)

		fails := report.TotalFails()
		if fails > 0 {
			log.S.Warn(
				"Quality control for the batch was not passed",
				logObject.Add("fails", fails).
					Add("details", report),
			)
		}
		totalFails += fails
	}

	if totalFails > 0 {
		log.S.Warn(
			"Quality control was not passed",
			logObject.Add("total_fails", totalFails),
		)
		return r.standby(ctx)
	}

	log.S.Info("Quality control has successfully been passed", logObject)
	return nil
}

// Generates QC report for the given batch.
func (r *Runner[S, R, P, Q]) qcReport(
	results []S,
	processingTime time.Duration,
) (report rd.QcReport) {
	logObject := log.L().Tag(log.LogTagQualityControl)
	numRequests := len(results)
	numSuccesses := util.Reduce(
		util.Map(results, func(res S) int {
			return res.GetStatusCode()
		}),
		0,
		func(acc int, v int) int {
			if v == 200 {
				return acc + 1
			}
			return acc
		},
	)

	// NOTE(nrydanov): Case 1. Batch processing takes too much time
	if processingTime > time.Duration(
		config.C.QualityControl.BatchTimeLimit,
	)*time.Second {
		log.S.Info(
			"Batch processing took longer than it should.",
			logObject.Add("expected", config.C.QualityControl.BatchTimeLimit).
				Add("elapsed", processingTime),
		)
		report.TimeLimitExceeded = true
	}

	// NOTE(nrydanov): Case 2. Too many requests ends with errors
	if numSuccesses < int(
		float64(numRequests)*config.C.QualityControl.SuccessThreshold,
	) {
		log.S.Info(
			"There were too many 5xx errors from the API.",
			logObject.Add("num_successes", numSuccesses).
				Add("num_requests", numRequests),
		)
		report.TooManyErrors = true
	}

	return report
}
