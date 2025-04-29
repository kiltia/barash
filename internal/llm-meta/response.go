package llmmeta

import (
	"encoding/json"
	"math"
	"time"

	"orb/runner/pkg/config"

	sf "github.com/sa-/slicefunk"
)

type LlmTaskResponse struct {
	Error    ErrorDetails `json:"component_error"`
	FinalUrl string       `json:"final_url"`
	Result   *struct {
		Score     float64   `json:"score"`
		MatchMask MatchMask `json:"match_mask"`
	} `json:"result"`
	DebugInfo DebugInfo `json:"debug_info"`
}

type ErrorDetails struct {
	ErrorRepr string `json:"error_repr"`
	ErrorType string `json:"error_type"`
	Code      string `json:"code"`
	Reason    string `json:"reason"`
}

type DebugInfo struct {
	CrawlerDebug struct {
		CrawlerErrors []map[string]any `json:"crawler_service_errors"`
		CrawlFails    []map[string]any `json:"crawl_parse_fails"`
		CrawledPages  []map[string]any `json:"crawled_pages"`
		PageStats     PageStats        `json:"page_stats"`
	} `json:"crawler_debug"`
	GeminiDebug struct {
		TokenCount struct {
			Discovery int `json:"discovery"`
			Task      int `json:"task"`
		} `json:"token_count"`
		Discovery []any `json:"discovery"`
		Task      []any `json:"task"`
	} `json:"gemini_debug"`
	MetricsDebug map[string]MetricEntry `json:"metrics_debug"`
}

type MetricEntry struct {
	ResponseTime float32 `json:"response_time"`
	StatusCode   uint16  `json:"status_code"`
}

type PageStats struct {
	Fails     int `json:"fails"`
	Errors    int `json:"errors"`
	Successes int `json:"successes"`
}

type MatchMask struct {
	Name                 MatchMaskEntry `json:"name"`
	Address1             MatchMaskEntry `json:"address1"`
	Address2             MatchMaskEntry `json:"address2"`
	City                 MatchMaskEntry `json:"city"`
	State                MatchMaskEntry `json:"state"`
	Country              MatchMaskEntry `json:"country"`
	Zip                  MatchMaskEntry `json:"zip"`
	DomainNameSimilarity struct {
		Reasoning string
		Value     float64
	} `json:"domain_name_similarity"`
}

type MatchMaskEntry struct {
	Reasoning string `json:"reasoning"`
	Value     string `json:"value"`
}

func serializeMap(m map[string]any) string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

func (response LlmTaskResponse) IntoStored(
	params LlmTaskParams,
	n int,
	url string,
	body map[string]any,
	status int,
	timeElapsed time.Duration,
) LlmTaskStoredResult {
	debugInfo := response.DebugInfo
	pageStats := response.DebugInfo.CrawlerDebug.PageStats
	crawlerDebug := debugInfo.CrawlerDebug
	result := response.Result

	var score float64
	if result == nil {
		score = math.NaN()
	} else {
		score = result.Score
	}

	ts := time.Now()

	convertMetrics := func(metrics map[string]MetricEntry) (
		map[string]float32,
		map[string]uint16,
	) {
		responseTimes := make(map[string]float32)
		responseCodes := make(map[string]uint16)

		for key, metric := range metrics {
			responseTimes[key] = metric.ResponseTime
			responseCodes[key] = metric.StatusCode
		}

		return responseTimes, responseCodes
	}

	responseTimes, responseCodes := convertMetrics(
		response.DebugInfo.MetricsDebug,
	)

	stored := LlmTaskStoredResult{
		Duns:            params.Duns,
		IsActive:        true,
		Url:             params.Url,
		FinalUrl:        response.FinalUrl,
		VerificationUrl: url,
		StatusCode:      int32(status),
		Error:           response.Error.Reason,
		ErrorCode:       response.Error.Code,
		ErrorType:       response.Error.ErrorType,
		ErrorRepr:       response.Error.ErrorRepr,
		AttemptsNumber:  int32(n),
		CrawlerErrors: sf.Map(
			crawlerDebug.CrawlerErrors,
			serializeMap,
		),
		CrawlFails:    sf.Map(crawlerDebug.CrawlFails, serializeMap),
		CrawledPages:  sf.Map(crawlerDebug.CrawledPages, serializeMap),
		NumErrors:     int32(pageStats.Errors),
		NumFails:      int32(pageStats.Fails),
		NumSuccesses:  int32(pageStats.Successes),
		ResponseTimes: responseTimes,
		ResponseCodes: responseCodes,
		Score:         score,
		Tag:           config.C.Run.Tag,
		Timestamp:     ts,
	}

	if result != nil {
		stored.MmName = result.MatchMask.Name.Value
		stored.MmAddress1 = result.MatchMask.Address1.Value
		stored.MmAddress2 = result.MatchMask.Address2.Value
		stored.MmCity = result.MatchMask.City.Value
		stored.MmState = result.MatchMask.State.Value
		stored.MmCountry = result.MatchMask.Country.Value
		stored.MmDomainNameSimilarity = result.MatchMask.DomainNameSimilarity.Value
		stored.MmZip = result.MatchMask.Zip.Value
	}

	return stored
}
