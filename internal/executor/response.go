package executor

import (
	"encoding/json"
	"time"

	"github.com/kiltia/runner/internal/crawler"
	"github.com/kiltia/runner/pkg/runner"
)

type PartialScrapeResponse struct {
	OriginalURL       string                    `json:"original_url"`
	FinalURL          string                    `json:"final_url"`
	TaskResult        json.RawMessage           `json:"task_result"`
	Status            int16                     `json:"status"`
	ResponseSize      int64                     `json:"response_size"`
	PartialParsedData crawler.PartialParsedData `json:"parsed"`
}

type ExecutorResponse struct {
	Responses  []PartialScrapeResponse `json:"responses"`
	Error      error                   `json:"error,omitempty"`
	TaskResult json.RawMessage         `json:"task_result,omitempty"`
	crawler.PartialErrorInfo
	Result Result `json:"result"`
}

type Result struct {
	Reduced json.RawMessage `json:"reduced"`
}

func (resp ExecutorResponse) IntoStored(
	req runner.ServiceRequest[ExecutorParams],
	err error,
	n int,
	status int,
	timeElapsed time.Duration,
	tag string,
) ExecutorResult {
	urls := map[string]struct{}{}
	for _, r := range resp.Responses {
		for _, u := range r.PartialParsedData.Urls {
			urls[u.URL] = struct{}{}
		}
	}
	urlSlice := make([]string, 0, len(urls))
	for u := range urls {
		urlSlice = append(urlSlice, u)
	}
	return ExecutorResult{
		URL:            req.Params.URL,
		RequestLink:    req.GetRequestLink(),
		StatusCode:     status,
		Error:          resp.Reason,
		ErrorType:      resp.ErrorType,
		ErrorCode:      resp.Code,
		AttemptsNumber: uint8(n),
		TaskResult:     resp.Result.Reduced,
		Urls:           urlSlice,
		TimeElapsed:    timeElapsed.Seconds(),
		Tag:            tag,
		Timestamp:      time.Now(),
	}
}
