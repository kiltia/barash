package runner

import "github.com/go-resty/resty/v2"

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

func (r *RetryTracker) Responses() []*resty.Response {
	responses := make([]*resty.Response, len(r.attempts))
	for i, attempt := range r.attempts {
		responses[i] = attempt.Response
	}
	return responses
}

func (r *RetryTracker) Attempts() []AttemptData {
	attempts := make([]AttemptData, len(r.attempts))
	copy(attempts, r.attempts)
	return attempts
}
