package meta

import (
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"time"

	"orb/runner/pkg/config"

	sf "github.com/sa-/slicefunk"
)

type ErrorDetails struct {
	ErrorRepr string `json:"error_repr"`
	ErrorType string `json:"error_type"`
	Code      string `json:"code"`
	Reason    string `json:"reason"`
}

// Response from the Meta API endpoint.
type VerifyResponse struct {
	Score     *float64     `json:"score"`
	Error     ErrorDetails `json:"component_error"`
	FinalUrl  string       `json:"final_url"`
	MatchMask MatchMask    `json:"match_mask"`
	DebugInfo DebugInfo    `json:"debug_info"`
}

func serializeMap(m map[string]any) string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

// Implement the [rinterface.Response] interface.
func (response VerifyResponse) IntoStored(
	params VerifyParams,
	n int,
	url string,
	body map[string]any,
	status int,
	timeElapsed time.Duration,
) VerifyResult {
	debugInfo := response.DebugInfo
	pageStats := response.DebugInfo.CrawlerDebug.PageStats
	crawlerDebug := debugInfo.CrawlerDebug
	matchMaskSummary := response.MatchMask.MatchMaskSummary

	var score float64
	if response.Score == nil {
		score = math.NaN()
	} else {
		score = *response.Score
	}

	ts := time.Now()
	var correctedTs time.Time

	convertMetrics := func(metrics MetricsDebug) (map[string]float32, map[string]uint16) {
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

	// NOTE(nrydanov): Need to replace with certain error code when we'll
	// determine it.
	if strings.Contains(strings.ToLower(response.Error.Code), "timeout") {
		// NOTE(nrydanov): This is a hack to avoid sitations when
		// too many potential timeouts are present in batch.
		seconds := rand.Intn(int(config.C.Run.MaxCorrection.Seconds()))
		correctedTs = ts.Add(
			time.Duration(
				seconds,
			) * time.Second,
		)
	} else {
		correctedTs = ts
	}

	return VerifyResult{
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
		CrawlFails:   sf.Map(crawlerDebug.CrawlFails, serializeMap),
		CrawledPages: sf.Map(crawlerDebug.CrawledPages, serializeMap),
		NumErrors:    int32(pageStats.Errors),
		NumFails:     int32(pageStats.Fails),
		NumSuccesses: int32(pageStats.Successes),
		Features: serializeMap(
			response.DebugInfo.FeatureExtractorDebug.Features,
		),
		MatchMaskDetails: serializeMap(
			response.MatchMask.MatchMaskDetails,
		),
		MmName:                 matchMaskSummary.Name,
		MmAddress1:             matchMaskSummary.Address1,
		MmAddress2:             matchMaskSummary.Address2,
		MmCity:                 matchMaskSummary.City,
		MmState:                matchMaskSummary.State,
		MmCountry:              matchMaskSummary.Country,
		MmDomainNameSimilarity: matchMaskSummary.DomainNameSimilarity,
		ResponseTimes:          responseTimes,
		ResponseCodes:          responseCodes,
		Score:                  score,
		Tag:                    config.C.Run.Tag,
		Timestamp:              ts,
		CorrTs:                 correctedTs,
	}
}

/* Below are the nested data structures. */

type FeatureExtractorDebug struct {
	Features map[string]any `json:"features"`
}

type MetricEntry struct {
	ResponseTime float32 `json:"response_time"`
	StatusCode   uint16  `json:"status_code"`
}

type MetricsDebug = map[string]MetricEntry

type DebugInfo struct {
	CrawlerDebug          CrawlerDebug          `json:"crawler_debug"`
	FeatureExtractorDebug FeatureExtractorDebug `json:"fe_debug"`
	MetricsDebug          MetricsDebug          `json:"metrics_debug"`
}

type CrawlerDebug struct {
	CrawlerErrors []map[string]any `json:"crawler_service_errors"`
	CrawlFails    []map[string]any `json:"crawl_parse_fails"`
	CrawledPages  []map[string]any `json:"crawled_pages"`
	PageStats     PageStats        `json:"page_stats"`
}

type PageStats struct {
	Fails     int `json:"fails"`
	Errors    int `json:"errors"`
	Successes int `json:"successes"`
}

type MatchMask struct {
	MatchMaskSummary MatchMaskSummary `json:"match_mask_summary"`
	MatchMaskDetails map[string]any   `json:"match_mask_details"`
}

type MatchMaskSummary struct {
	Name                 string  `json:"name"`
	Address1             string  `json:"address1"`
	Address2             string  `json:"address2"`
	City                 string  `json:"city"`
	State                string  `json:"state"`
	Country              string  `json:"country"`
	DomainNameSimilarity float64 `json:"domain_name_similarity"`
}
