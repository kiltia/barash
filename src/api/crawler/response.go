package crawler

import "time"

type CrawlerResponse struct {
	OriginalUrl  string            `json:"original_url"`
	FinalUrl     string            `json:"final_url"`
	Status       int16             `json:"status"`
	ResponseSize int64             `json:"response_size"`
	HeadlessUsed bool              `json:"headless_used"`
	Parsed       PartialParsedData `json:"parsed"`
}

func (resp CrawlerResponse) IntoStored(
	params CrawlerRequest,
	n int,
	url string,
	status int,
	timeElapsed time.Duration,
) CrawlingResult {
	return CrawlingResult{
		AttemptsNumber:   n,
		CrawlerParams:    params,
		CrawlingResponse: &resp,
		RequestLink:      url,
		StatusCode:       status,
		TimeElapsed:      timeElapsed,
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
