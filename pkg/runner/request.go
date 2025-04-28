package runner

import (
	"fmt"
	"net/url"

	ri "orb/runner/pkg/runner/interface"
	"orb/runner/pkg/util"
)

type GetRequest[P ri.StoredParams] struct {
	Host        string
	Port        string
	Method      string
	Params      P
	ExtraParams map[string]string

	cachedRequestLink *string
}

func (req *GetRequest[P]) GetRequestLink() (string, error) {
	if req.cachedRequestLink != nil {
		return *req.cachedRequestLink, nil
	}

	baseURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", req.Host, req.Port),
		Path:   req.Method,
	}
	params := url.Values{}
	paramsMap, err := util.ObjectToMap(req.Params)
	if err != nil {
		return "", fmt.Errorf("unable to create request link, reason: %v", err)
	}
	for field, value := range paramsMap {
		if value != nil && *value != "" {
			params.Add(field, *value)
		}
	}
	for field, value := range req.ExtraParams {
		params.Add(field, value)
	}
	baseURL.RawQuery = params.Encode()
	urlString := baseURL.String()

	req.cachedRequestLink = &urlString
	return urlString, nil
}
