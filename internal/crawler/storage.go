package crawler

import (
	"fmt"
	"time"

	"orb/runner/pkg/config"
)

type CrawlerResult struct {
	Url               string    `ch:"url"`
	RequestLink       string    `ch:"request_link"`
	CrawlerStatusCode uint16    `ch:"crawler_status_code"`
	SiteStatusCode    uint16    `ch:"site_status_code"`
	Error             string    `ch:"error"`
	ErrorType         string    `ch:"error_type"`
	ErrorCode         string    `ch:"error_code"`
	AttemptsNumber    uint8     `ch:"attempts_number"`
	OriginalUrl       string    `ch:"original_url"`
	FinalUrl          string    `ch:"final_url"`
	ResponseSize      int64     `ch:"response_size"`
	HeadlessUsed      bool      `ch:"headless_used"`
	Urls              []string  `ch:"urls"`
	TimeElapsed       float64   `ch:"time_elapsed"`
	Tag               string    `ch:"tag"`
	Timestamp         time.Time `ch:"ts"`
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlerResult) GetCreateQuery() string {
	query := fmt.Sprintf(
		`
        CREATE TABLE %s
        (
            url String,
            request_link String,
            crawler_status_code UInt16,
            site_status_code UInt16,
            error String,
            error_type String,
            error_code String,
            attempts_number UInt8,
            original_url String,
            final_url String,
            response_size Int64,
            headless_used Bool,
            urls Array(String),
            time_elapsed Float64,
            tag String,
            ts DateTime64
        )
        ENGINE = MergeTree
        ORDER BY ts
    `, config.C.Run.InsertionTableName)
	return query
}
