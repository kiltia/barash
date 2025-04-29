package runner

import (
	"fmt"
	"net/url"

	"orb/runner/pkg/config"
)

type ServiceRequest[P StoredParams] struct {
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

	params := ObjectToParams(req.Params)
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

	body := ObjectToBody(req.Params)

	req.cachedRequestBody = body
	return body
}
