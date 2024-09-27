package crawler

import (
	"fmt"
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
	query := fmt.Sprintf(
		`
        INSERT INTO %s VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, fromUnixTimestamp64Micro(?)
        )
    `, config.C.Run.InsertionTableName)
	return query
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlerResult) GetStatusCode() int {
	return r.StatusCode
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlerResult) GetCreateQuery() string {
	query := fmt.Sprintf(
		`
        CREATE TABLE %s
        (
            url String,
            request_link String,
            crawler_status_code Int16,
            site_status_code Int16,
            error String,
            error_type String,
            error_code String,
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
    `, config.C.Run.InsertionTableName)
	return query
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
		r.CrawlerResponse.ErrorInfo.ErrorType,
		r.CrawlerResponse.ErrorInfo.Code,
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
