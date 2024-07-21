package rdata

import "orb/runner/pkg/util"

type QcReport struct {
	// It took way too long to process this batch.
	TimeLimitExceeded bool `json:"time_limit_exeeded"`
	// Too many errors from the API.
	TooManyErrors bool `json:"too_many_errors"`
	// User-defined criteria.
	Extra map[string]bool `json:"extra"`
}

func (r *QcReport) TotalFails() (total int) {
	if r.TimeLimitExceeded {
		total += 1
	}
	if r.TooManyErrors {
		total += 1
	}

	// NOTE(evgenymng): Go'ing functional
	return util.Reduce(
		util.Values(r.Extra),
		total,
		func(acc int, f bool) int {
			if f {
				return acc + 1
			}
			return acc
		},
	)
}
