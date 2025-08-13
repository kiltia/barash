package runner

import "resty.dev/v3"

type AttemptData struct {
	Response *resty.Response
	Error    error
}

type RetryTracker struct {
	attempts []AttemptData
}

func (r *RetryTracker) Add(resp *resty.Response, err error) {
	r.attempts = append(r.attempts, AttemptData{
		Response: resp,
		Error:    err,
	})
}

func (r *RetryTracker) Attempts() []AttemptData {
	attempts := make([]AttemptData, len(r.attempts))
	copy(attempts, r.attempts)
	return attempts
}
