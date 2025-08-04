package crawler

import (
	"time"
)

type PartialErrorInfo struct {
	Reason    string `json:"reason"`
	Code      string `json:"code"`
	ErrorType string `json:"error_type"`
}

type CrawlerResponse struct {
	OriginalUrl  string            `json:"original_url"`
	FinalUrl     string            `json:"final_url"`
	Status       int16             `json:"status"`
	ResponseSize int64             `json:"response_size"`
	HeadlessUsed bool              `json:"headless_used"`
	Parsed       PartialParsedData `json:"parsed"`
	ErrorInfo    PartialErrorInfo  `json:"error"`
}

func (resp CrawlerResponse) IntoStored(
	params CrawlerParams,
	n int,
	url string,
	body map[string]any,
	status int,
	timeElapsed time.Duration,
	tag string,
) CrawlerResult {
	var urls []string
	for i := range resp.Parsed.Urls {
		urls = append(urls, resp.Parsed.Urls[i].URL)
	}
	return CrawlerResult{
		Url:               params.Url,
		RequestLink:       url,
		CrawlerStatusCode: uint16(status),
		SiteStatusCode:    uint16(resp.Status),
		Error:             resp.ErrorInfo.Reason,
		ErrorType:         resp.ErrorInfo.ErrorType,
		ErrorCode:         resp.ErrorInfo.Code,
		AttemptsNumber:    uint8(n),
		OriginalUrl:       resp.OriginalUrl,
		FinalUrl:          resp.FinalUrl,
		ResponseSize:      resp.ResponseSize,
		HeadlessUsed:      resp.HeadlessUsed,
		Urls:              urls,
		TimeElapsed:       timeElapsed.Seconds(),
		Tag:               tag,
		Timestamp:         time.Now(),
	}
}

/* Below are the nested data structures. */

type AttributeInfo struct {
	Rel      *string `json:"rel"`
	HrefLang *string `json:"hreflang"`
	Title    *string `json:"title"`
	Type     *string `json:"type"`
	Alt      *string `json:"alt"`
	Sizes    *string `json:"sizes"`
}

type CrawledUrl struct {
	Original   string        `json:"original"`
	URL        string        `json:"url"`
	Type       string        `json:"type"`
	TagName    string        `json:"tag_name"`
	AnchorText *string       `json:"anchor_text"`
	Attributes AttributeInfo `json:"attributes"`
}

type PartialParsedData struct {
	Urls []CrawledUrl `json:"urls"`
}
