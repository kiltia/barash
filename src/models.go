package main

import (
	"fmt"
	"net/url"
)

type JSONString string

func (js *JSONString) UnmarshalJSON(b []byte) error {
	*js = JSONString(b)
	return nil
}

type VerifyGetRequest struct {
	Host         string
	Port         string
	Method       string
	VerifyParams VerifyParams
}

type VerifyParams struct {
	Duns                  string  `json:"duns"                    csv:"duns"`
	Url                   string  `json:"url"                     csv:"url"`
	DomainCrawlerStrategy *string `json:"domain_crawler_strategy"`
	Name                  *string `json:"name"                    csv:"name"`
	LocAddress1           *string `json:"loc_address1"            csv:"loc_address1"`
	LocAddress2           *string `json:"loc_address2"            csv:"loc_address2"`
	MailAddress1          *string `json:"mail_address1"           csv:"mail_address1"`
	MailAddress2          *string `json:"mail_address2"           csv:"mail_address2"`
	MailCity              *string `json:"mail_city"               csv:"mail_city"`
	LocCity               *string `json:"loc_city"                csv:"loc_city"`
	LocState              *string `json:"loc_state"               csv:"loc_state"`
	MailState             *string `json:"mail_state"              csv:"mail_state"`
	MailZip               *string `json:"mail_zip"                csv:"mail_zip"`
	LocZip                *string `json:"loc_zip"                 csv:"loc_zip"`
	MailCountry           *string `json:"mail_country"            csv:"mail_country"`
	LocCountry            *string `json:"loc_country"             csv:"loc_country"`
}

type VerificationResponse struct {
	Score     *float64  `json:"score"`
	Error     *string   `json:"component_error"`
	MatchMask MatchMask `json:"match_mask"`
	DebugInfo DebugInfo `json:"debug"`
}

type DebugInfo struct {
	Features     *JSONString  `json:"features"`
	CrawlerDebug CrawlerDebug `json:"crawler_debug"`
}

type CrawlerDebug struct {
	CrawlerErrors []*JSONString `json:"crawler_errors"`
	CrawlFails    []*JSONString `json:"crawl_fails"`
	CrawledPages  []*JSONString `json:"crawled_pages"`
	FailStatus    *string       `json:"fail_status"`
	PageStats     PageStats     `json:"page_stats"`
}

type PageStats struct {
	Fails     *int `json:"fails"`
	Errors    *int `json:"errors"`
	Successes *int `json:"successes"`
}

type MatchMask struct {
	MatchMaskSummary MatchMaskSummary `json:"match_mask_summary"`
	MatchMaskDetails *JSONString      `json:"match_mask_details"`
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

type VerificationResult struct {
	StatusCode           int
	VerifyParams         VerifyParams
	VerificationLink     string
	VerificationResponse *VerificationResponse
}

func (verifyGetRequest VerifyGetRequest) CreateVerifyGetRequestLink(extraParams map[string]string) (string, error) {
	baseURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", verifyGetRequest.Host, verifyGetRequest.Port),
		Path:   verifyGetRequest.Method,
	}
	params := url.Values{}
	paramsMap, err := structToMap(verifyGetRequest.VerifyParams)
	if err != nil {
		return "", fmt.Errorf("Unable to create verify link. Reason: %v", err)
	}
	for field, value := range paramsMap {
		if value != nil {
			params.Add(field, *value)
		}
	}
	for field, value := range extraParams {
		params.Add(field, value)
	}
	baseURL.RawQuery = params.Encode()
	urlString := baseURL.String()
	return urlString, nil
}

func NewVerifyGetRequest(
	host string,
	port string,
	method string,
	verifyParams VerifyParams,
) *VerifyGetRequest {
	return &VerifyGetRequest{host, port, method, verifyParams}
}
