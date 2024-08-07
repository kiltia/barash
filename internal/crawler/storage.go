package crawler

import (
	"strings"
	"time"

	"orb/runner/pkg/config"
)

type CrawlerResult struct {
	StatusCode      int
	CrawlerParams   CrawlerParams
	RequestLink     string
	AttemptsNumber  int
	TimeElapsed     time.Duration
	CrawlerResponse *CrawlerResponse
	Timestamp       time.Time
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlerResult) GetInsertQuery() string {
	return `
        INSERT INTO crawler VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, fromUnixTimestamp64Micro(?)

        )
    `
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlerResult) GetStatusCode() int {
	return r.StatusCode
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlerResult) GetCreateQuery() string {
	return `
        CREATE TABLE wv.crawler
        (
            url String,
            request_link String,
            crawler_status_code Int16,
            site_status_code Int16,
            error String,
            attempts Int16,
            original_url String,
            final_url String,
            response_size Int128,
            headless_used Bool,
            urls Array(String),
            time_elapsed Int32,
            tag String,
            ts DateTime64
        )
        ENGINE = MergeTree
        ORDER BY ts
    `
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlerResult) AsArray() []any {
	crawlingParams := r.CrawlerParams
	response := r.CrawlerResponse

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
		response.Status,
		r.CrawlerResponse.ErrorInfo.Reason,
		r.AttemptsNumber,
		response.OriginalUrl,
		response.FinalUrl,
		response.ResponseSize,
		response.HeadlessUsed,
		urls,
		r.TimeElapsed.Abs().Milliseconds(),
		config.C.Run.Tag,
		r.Timestamp.UnixMicro(),
	}
}
