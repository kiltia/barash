package runner

import (
	"fmt"
	"net/url"

	"orb/runner/pkg/config"
	ri "orb/runner/pkg/runner/interface"
	"orb/runner/pkg/util"
)

type ServiceRequest[P ri.StoredParams] struct {
	Host        string
	Port        string
	Endpoint    string
	Method      config.RunnerHttpMethod
	Params      P
	ExtraParams map[string]string

	cachedRequestLink string
	cachedRequestBody map[string]any
}

func (req *ServiceRequest[P]) GetRequestLink() string {
	if req.cachedRequestLink != "" {
		return req.cachedRequestLink
	}

	baseURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", req.Host, req.Port),
		Path:   req.Endpoint,
	}

	queryParams := util.ObjectToParams(req.Params)

	params := url.Values{}
	for field, value := range queryParams {
		params.Add(field, value)
	}
	for field, value := range req.ExtraParams {
		params.Add(field, value)
	}

	baseURL.RawQuery = params.Encode()
	urlString := baseURL.String()

	req.cachedRequestLink = urlString
	return urlString
}

func (req *ServiceRequest[P]) GetRequestBody() map[string]any {
	if req.cachedRequestBody != nil {
		return req.cachedRequestBody
	}

	body := util.ObjectToBody(req.Params)

	req.cachedRequestBody = body
	return body
}
