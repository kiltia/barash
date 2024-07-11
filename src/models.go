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

type GetRequest[P ParamsType] struct {
	Host   string
	Port   string
	Method string
	Params P
}

type VerifyParams struct {
	Duns         string  `json:"duns"          ch:"duns"`
	Url          string  `json:"url"           ch:"url"`
	Name         *string `json:"name"          ch:"name"`
	LocAddress1  *string `json:"loc_address1"  ch:"loc_address1"`
	LocAddress2  *string `json:"loc_address2"  ch:"loc_address2"`
	MailAddress1 *string `json:"mail_address1" ch:"mail_address1"`
	MailAddress2 *string `json:"mail_address2" ch:"mail_address2"`
	MailCity     *string `json:"mail_city"     ch:"mail_city"`
	LocCity      *string `json:"loc_city"      ch:"loc_city"`
	LocState     *string `json:"loc_state"     ch:"loc_state"`
	MailState    *string `json:"mail_state"    ch:"mail_state"`
	MailZip      *string `json:"mail_zip"      ch:"mail_zip"`
	LocZip       *string `json:"loc_zip"       ch:"loc_zip"`
	MailCountry  *string `json:"mail_country"  ch:"mail_country"`
	LocCountry   *string `json:"loc_country"   ch:"loc_country"`
}

type VerificationResponse struct {
	Score     *float64  `json:"score"`
	Error     *string   `json:"component_error"`
	FinalUrl  *string   `json:"final_url"`
	MatchMask MatchMask `json:"match_mask"`
	DebugInfo DebugInfo `json:"debug_info"`
}

func (response VerificationResponse) IntoWith(
	params VerifyParams,
	n int,
	url string,
	status int,
) VerificationResult {
	return VerificationResult{
		AttemptsNumber:       n,
		VerifyParams:         params,
		VerificationResponse: &response,
		VerificationLink:     url,
		StatusCode:           status,
	}
}

type DebugInfo struct {
	// TODO(nrydanov): Fix features (more information from Sergey Okunkov)
	Features     *string      `json:"features"`
	CrawlerDebug CrawlerDebug `json:"crawler_debug"`
}

type CrawlerDebug struct {
	CrawlerErrors []*JSONString `json:"crawler_service_errors"`
	CrawlFails    []*JSONString `json:"crawl_parse_fails"`
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
	AttemptsNumber       int
	VerificationResponse *VerificationResponse
}

// Implement the [StoredValueType] interface.
func (r VerificationResult) GetInsertQuery() string {
	return `INSERT INTO master VALUES (
        ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
        ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
        ?, ?, ?, ?, ?, ?, now()
    )`
}

// Implement the [StoredValueType] interface.
func (r VerificationResult) GetStatusCode() int {
	return r.StatusCode
}

// Implement the [StoredValueType] interface.
func (r VerificationResult) GetSelectQuery() string {
	return `
        with last as (
            select duns, url, max(ts) as max_ts
            from wv.master
            where is_active = True
            group by duns, url
        ),
        batch as (
            select duns, url, max_ts
            from last
            where max_ts < (now() - toIntervalDay(%d))
            limit %d
        ),
        final as (
            select
                batch.duns as duns,
                batch.url as url,
                gdmi.name,
                gdmi.loc_address1, gdmi.loc_address2,
                gdmi.loc_city, gdmi.loc_state,
                gdmi.loc_zip, gdmi.loc_country,
                gdmi.mail_address1, gdmi.mail_address2,
                gdmi.mail_city, gdmi.mail_state,
                gdmi.mail_zip, gdmi.mail_country
            from wv.gdmi_compact gdmi
            inner join batch using (duns)
            where gdmi.duns != '' and batch.url != ''
            order by cityHash64(batch.duns, batch.url)
        )
        select * from final
    `
}

// Implement the [StoredValueType] interface.
func (r VerificationResult) GetCreateQuery() string {
	// TODO(evgenymng): Return something
	return ""
}

// Implement the [StoredValueType] interface.
func (r VerificationResult) AsArray() []any {
	verifyParams := r.VerifyParams
	response := r.VerificationResponse
	debugInfo := r.VerificationResponse.DebugInfo
	pageStats := r.VerificationResponse.DebugInfo.CrawlerDebug.PageStats
	crawlerDebug := debugInfo.CrawlerDebug
	MatchMaskSummary := response.MatchMask.MatchMaskSummary

	return []any{
		verifyParams.Duns,
		true,
		verifyParams.Url,
		r.VerificationLink,
		r.StatusCode,
		response.Error,
		crawlerDebug.FailStatus,
		r.AttemptsNumber,
		crawlerDebug.CrawlerErrors,
		crawlerDebug.CrawlFails,
		crawlerDebug.CrawledPages,
		pageStats.Errors,
		pageStats.Fails,
		pageStats.Successes,
		debugInfo.Features,
		response.MatchMask.MatchMaskDetails,
		MatchMaskSummary.Name,
		MatchMaskSummary.Address1,
		MatchMaskSummary.Address2,
		MatchMaskSummary.City,
		MatchMaskSummary.State,
		MatchMaskSummary.Country,
		MatchMaskSummary.DomainNameSimilarity,
		response.FinalUrl,
		response.Score,
	}
}

func (req GetRequest[P]) CreateGetRequestLink(
	extraParams map[string]string,
) (string, error) {
	baseURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", req.Host, req.Port),
		Path:   req.Method,
	}
	params := url.Values{}
	paramsMap, err := structToMap(req.Params)
	if err != nil {
		return "", fmt.Errorf("Unable to create request link. Reason: %v", err)
	}
	for field, value := range paramsMap {
		if value != nil && *value != "" {
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

func NewGetRequest[P ParamsType](
	host string,
	port string,
	method string,
	verifyParams P,
) *GetRequest[P] {
	return &GetRequest[P]{host, port, method, verifyParams}
}
