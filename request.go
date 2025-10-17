package barash

import (
	"net/url"

	"github.com/kiltia/barash/config"
)

type ServiceRequest[P StoredParams] struct {
	RequestURL url.URL
	Method     config.RunnerHTTPMethod
	Params     P

	cachedRequestLink string
	cachedRequestBody []byte
}

func (req *ServiceRequest[P]) GetRequestLink() string {
	if req.cachedRequestLink != "" {
		return req.cachedRequestLink
	}

	baseURL := req.RequestURL

	query := baseURL.Query()
	params := ObjectToParams(req.Params)
	for k, v := range params {
		for _, vv := range v {
			query.Add(k, vv)
		}
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
