package runner

import (
	"fmt"
	"net/url"

	"github.com/kiltia/runner/pkg/config"
)

type ServiceRequest[P StoredParams] struct {
	Host        string
	Port        string
	Endpoint    string
	Scheme      string
	Method      config.RunnerHTTPMethod
	Params      P
	ExtraParams map[string]string

	cachedRequestLink string
	cachedRequestBody []byte
}

func (req *ServiceRequest[P]) GetRequestLink() string {
	if req.cachedRequestLink != "" {
		return req.cachedRequestLink
	}

	baseURL := &url.URL{
		Scheme: req.Scheme,
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

func (req *ServiceRequest[P]) GetRequestBody() []byte {
	if req.cachedRequestBody != nil {
		return req.cachedRequestBody
	}

	body := ObjectToBody(&req.Params)

	req.cachedRequestBody = body
	return body
}
