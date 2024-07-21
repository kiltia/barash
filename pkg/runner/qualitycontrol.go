package runner

import (
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	rd "orb/runner/pkg/runner/data"
	"orb/runner/pkg/util"
)

// Performs quality control of a processed batch.
func (r *Runner[S, R, P, Q]) qualityControl(
	results []S,
	processingTime time.Duration,
) (report rd.QcReport) {
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
		log.S.Infow(
			"Batch processing takes longer than it should.",
			"num_successes", numSuccesses,
			"num_requests", numRequests,
			"tag", log.TagQualityControl,
		)
		report.TimeLimitExceeded = true
	}

	// NOTE(nrydanov): Case 2. Too many requests ends with errors
	if numSuccesses < int(
		float64(numRequests)*config.C.QualityControl.SuccessThreshold,
	) {
		log.S.Infow(
			"Too many 5xx errors from the API.",
			"tag", log.TagQualityControl,
		)
		report.TooManyErrors = true
	}

	return report
}
