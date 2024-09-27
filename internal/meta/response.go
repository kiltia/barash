package meta

import (
	"time"

	"orb/runner/internal/common"
)

type ErrorDetails struct {
	ErrorRepr *string `json:"error_repr"`
	ErrorType *string `json:"error_type"`
	Code      *string `json:"code"`
	Reason    *string `json:"reason"`
}

// Response from the Meta API endpoint.
type VerifyResponse struct {
	Score     *float64     `json:"score"`
	Error     ErrorDetails `json:"component_error"`
	FinalUrl  *string      `json:"final_url"`
	MatchMask MatchMask    `json:"match_mask"`
	DebugInfo DebugInfo    `json:"debug_info"`
}

// Implement the [rinterface.Response] interface.
func (resp VerifyResponse) IntoStored(
	params VerifyParams,
	n int,
	url string,
	status int,
	timeElapsed time.Duration,
) VerifyResult {
	return VerifyResult{
		AttemptsNumber: n,
		VerifyParams:   params,
		MetaResponse:   resp,
		RequestLink:    url,
		StatusCode:     status,
		TimeElapsed:    timeElapsed,
		Timestamp:      time.Now(),
	}
}

/* Below are the nested data structures. */

type FeatureExtractorDebug struct {
	Features *common.JsonString `json:"features"`
}

type DebugInfo struct {
	CrawlerDebug          CrawlerDebug          `json:"crawler_debug"`
	FeatureExtractorDebug FeatureExtractorDebug `json:"fe_debug"`
}

type CrawlerDebug struct {
	CrawlerErrors []*common.JsonString `json:"crawler_service_errors"`
	CrawlFails    []*common.JsonString `json:"crawl_parse_fails"`
	CrawledPages  []*common.JsonString `json:"crawled_pages"`
	PageStats     PageStats            `json:"page_stats"`
}

type PageStats struct {
	Fails     *int `json:"fails"`
	Errors    *int `json:"errors"`
	Successes *int `json:"successes"`
}

type MatchMask struct {
	MatchMaskSummary MatchMaskSummary   `json:"match_mask_summary"`
	MatchMaskDetails *common.JsonString `json:"match_mask_details"`
}

type MatchMaskSummary struct {
	Name                 *string  `json:"name"`
	Address1             *string  `json:"address1"`
	Address2             *string  `json:"address2"`
	City                 *string  `json:"city"`
	State                *string  `json:"state"`
	Country              *string  `json:"country"`
	DomainNameSimilarity *float64 `json:"domain_name_similarity"`
}
