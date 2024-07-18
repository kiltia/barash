package crawler

import (
	"strings"
	"time"
)

type CrawlingResult struct {
	StatusCode       int
	CrawlerParams    CrawlerRequest
	RequestLink      string
	AttemptsNumber   int
	TimeElapsed      time.Duration
	CrawlingResponse *CrawlerResponse
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlingResult) GetInsertQuery() string {
	return `
        INSERT INTO crawler VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now()
        )
    `
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlingResult) GetStatusCode() int {
	return r.StatusCode
}

func (r CrawlingResult) GetUrl() string {
	return r.CrawlerParams.Url
}

func (p CrawlerRequest) GetUrl() string {
	return p.Url
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlingResult) GetCreateQuery() string {
	return `
        CREATE TABLE wv.crawler
        (
            url String,
            request_link String,
            status Int16,
            attempts Int16,
            original_url String,
            final_url String,
            status_code Int16,
            response_size Int128,
            headless_used Bool,
            urls Array(String),
            time_elapsed Int32,
            tag String,
            ts DateTime
        )
        ENGINE = MergeTree
        ORDER BY ts
    `
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlingResult) AsArray() []any {
	crawlingParams := r.CrawlerParams
	response := r.CrawlingResponse

	urls := []string{}

	for _, url := range response.Parsed.Urls {
		if url.TagName == "a" && url.Type == "href" &&
			(strings.HasPrefix(url.URL, "http") || strings.HasPrefix(url.URL, "https")) {
			urls = append(urls, url.URL)
		}
	}

	return []any{
		crawlingParams.Url,
		r.RequestLink,
		r.StatusCode,
		r.AttemptsNumber,
		response.OriginalUrl,
		response.FinalUrl,
		response.Status,
		response.ResponseSize,
		response.HeadlessUsed,
		urls,
		r.TimeElapsed.Abs().Milliseconds(),
	}
}
