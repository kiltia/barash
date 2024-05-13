package main

import (
	"fmt"
	"strings"
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
	Url                   string  `json:"url" csv:"url"`
	DomainCrawlerStrategy *string `json:"domain_crawler_strategy"`
	Name                  *string `json:"name" csv:"name"`
	LocAddress1           *string `json:"loc_address1" csv:"loc_address1"`
	LocAddress2           *string `json:"loc_address2" csv:"loc_address2"`
	MailAddress1          *string `json:"mail_address1" csv:"mail_address1"`
	MailAddress2          *string `json:"mail_address2" csv:"mail_address2"`
	MailCity              *string `json:"mail_city" csv:"mail_city"`
	LocCity               *string `json:"loc_city" csv:"loc_city"`
	LocState              *string `json:"loc_state" csv:"loc_state"`
	MailState             *string `json:"mail_state" csv:"mail_state"`
	MailZip               *string `json:"mail_zip" csv:"mail_zip"`
	LocZip                *string `json:"loc_zip" csv:"loc_zip"`
	MailCountry           *string `json:"mail_country" csv:"mail_country"`
	LocCountry            *string `json:"loc_country" csv:"loc_country"`
}

type VerificationResponse struct {
	Score     *float64  `json:"score"`
	Error     *string   `json:"component_error"`
	MatchMask MatchMask `json:"match_mask"`
	DebugInfo DebugInfo `json:"debug"`
}

type DebugInfo struct {
	Features     *JSONString `json:"features"`
	CrawlerDebug *JSONString `json:"crawler_debug"`
}

type MatchMask struct {
	MatchMaskSummary *JSONString `json:"match_mask_summary"`
	MatchMaskDetails *JSONString `json:"match_mask_details"`
}

type VerificationResult struct {
	StatusCode           int
	VerifyParams         VerifyParams
	VerificationLink     string
	VerificationResponse *VerificationResponse
}

func (verifyGetRequest VerifyGetRequest) CreateVerifyGetRequestLink() (string, error) {
	var url string = verifyGetRequest.Host + ":" + verifyGetRequest.Port + verifyGetRequest.Method
	var paramsString string = ""
	paramsMap, err := structToMap(verifyGetRequest.VerifyParams)
	for field, value := range paramsMap {
		if value != nil {
			paramsString += field + "=" + strings.ReplaceAll(*value, " ", "+") + "&"
		}
	}
	if err != nil || len(paramsString) == 0 {
		return "", fmt.Errorf("Unable to create verify link")
	}
	return url + "?" + paramsString[:len(paramsString)-1], nil
}

func NewVerifyGetRequest(host string, port string, method string, verifyParams VerifyParams) *VerifyGetRequest {
	return &VerifyGetRequest{host, port, method, verifyParams}
}
