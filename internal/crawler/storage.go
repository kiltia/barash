package crawler

import (
	"fmt"
	"time"

	"orb/runner/pkg/config"
)

type CrawlerResult struct {
	Url               string    `json:"url"                 ch:"url"`
	RequestLink       string    `json:"request_link"        ch:"request_link"`
	CrawlerStatusCode uint16    `json:"crawler_status_code" ch:"crawler_status_code"`
	SiteStatusCode    uint16    `json:"site_status_code"    ch:"site_status_code"`
	Error             string    `json:"error"               ch:"error"`
	ErrorType         string    `json:"error_type"          ch:"error_type"`
	ErrorCode         string    `json:"error_code"          ch:"error_code"`
	AttemptsNumber    uint8     `json:"attempts_number"     ch:"attempts_number"`
	OriginalUrl       string    `json:"original_url"        ch:"original_url"`
	FinalUrl          string    `json:"final_url"           ch:"final_url"`
	ResponseSize      int64     `json:"response_size"       ch:"response_size"`
	HeadlessUsed      bool      `json:"headless_used"       ch:"headless_used"`
	Urls              []string  `json:"urls"                ch:"urls"`
	TimeElapsed       int64     `json:"time_elapsed"        ch:"time_elapsed"`
	Tag               string    `json:"tag"                 ch:"tag"`
	Timestamp         time.Time `json:"ts"                  ch:"ts"`
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
            attempts_number Int32,
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
