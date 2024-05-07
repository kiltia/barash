package main

import (
	"fmt"
	"strings"
)

type VerifyGetRequest struct {
	Host         string
	Port         string
	Method       string
	VerifyParams VerifyParams
}

type VerifyParams struct {
	Url                   string  `json:"url"`
	DomainCrawlerStrategy *string `json:"domain_crawler_strategy"`
	Name                  *string `json:"name"`
	LocAddress1           *string `json:"loc_address1"`
	LocAddress2           *string `json:"loc_address2"`
	MailAddress1          *string `json:"mail_address1"`
	MailAddress2          *string `json:"mail_address2"`
	MailCity              *string `json:"mail_city"`
	LocCity               *string `json:"loc_city"`
	LocState              *string `json:"loc_state"`
	MailState             *string `json:"mail_state"`
	MailZip               *string `json:"mail_zip"`
	LocZip                *string `json:"loc_zip"`
	MailCountry           *string `json:"mail_country"`
	LocCountry            *string `json:"loc_country"`
}

func (verifyGetRequest VerifyGetRequest) CreateVerifyGetRequestLink() (string, error) {
	var url string = verifyGetRequest.Host + ":" + verifyGetRequest.Port + verifyGetRequest.Method
	var params_string string = ""
	params_map, err := structToMap(verifyGetRequest.VerifyParams)
	for field, value := range params_map {
		if value != nil {
			params_string += field + "=" + strings.ReplaceAll(*value, " ", "+") + "&"
		}
	}
	if err != nil || len(params_string) == 0 {
		return "", fmt.Errorf("Unable to create verify link")
	}
	return url + "?" + params_string[:len(params_string)-1], nil
}

func NewVerifyGetRequest(host string, port string, method string, verifyParams VerifyParams) *VerifyGetRequest {
	return &VerifyGetRequest{host, port, method, verifyParams}
}
