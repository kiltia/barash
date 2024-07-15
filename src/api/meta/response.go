package metaapi

import "orb/runner/src/api/common"

// Response from the Meta API endpoint.
type VerifyResponse struct {
	Score     *float64  `json:"score"`
	Error     *string   `json:"component_error"`
	FinalUrl  *string   `json:"final_url"`
	MatchMask MatchMask `json:"match_mask"`
	DebugInfo DebugInfo `json:"debug_info"`
}

// Implement the [rinterface.Response] interface.
func (resp VerifyResponse) IntoStored(
	params VerifyRequestParams,
	n int,
	url string,
	status int,
) VerificationResult {
	return VerificationResult{
		AttemptsNumber:       n,
		VerifyParams:         params,
		VerificationResponse: &resp,
		RequestLink:          url,
		StatusCode:           status,
	}
}

/* Below are the nested data structures. */

type DebugInfo struct {
	// TODO(nrydanov): Fix features (more information from Sergey Okunkov)
	Features     *string      `json:"features"`
	CrawlerDebug CrawlerDebug `json:"crawler_debug"`
}

type CrawlerDebug struct {
	CrawlerErrors []*common.JsonString `json:"crawler_service_errors"`
	CrawlFails    []*common.JsonString `json:"crawl_parse_fails"`
	CrawledPages  []*common.JsonString `json:"crawled_pages"`
	FailStatus    *string              `json:"fail_status"`
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
