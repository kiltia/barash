package main

import (
	"fmt"
	"net/url"
)

type GetRequest[P ParamsType] struct {
	Host   string
	Port   string
	Method string
	Params P
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
	params P,
) *GetRequest[P] {
	return &GetRequest[P]{host, port, method, params}
}
