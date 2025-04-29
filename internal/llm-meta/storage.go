package llmmeta

import (
	"fmt"
	"time"

	"orb/runner/pkg/config"
)

type LlmTaskStoredResult struct {
	Duns                   string             `ch:"duns"`
	IsActive               bool               `ch:"is_active"`
	Url                    string             `ch:"url"`
	VerificationUrl        string             `ch:"verification_url"`
	StatusCode             int32              `ch:"status_code"`
	Error                  string             `ch:"error"`
	ErrorCode              string             `ch:"error_code"`
	ErrorType              string             `ch:"error_type"`
	ErrorRepr              string             `ch:"error_repr"`
	AttemptsNumber         int32              `ch:"attempts_number"`
	CrawlerErrors          []string           `ch:"crawler_errors"`
	CrawlFails             []string           `ch:"crawl_fails"`
	CrawledPages           []string           `ch:"crawled_pages"`
	NumErrors              int32              `ch:"num_errors"`
	NumFails               int32              `ch:"num_fails"`
	NumSuccesses           int32              `ch:"num_successes"`
	MmName                 string             `ch:"mm_name"`
	MmAddress1             string             `ch:"mm_address1"`
	MmAddress2             string             `ch:"mm_address2"`
	MmCity                 string             `ch:"mm_city"`
	MmState                string             `ch:"mm_state"`
	MmCountry              string             `ch:"mm_country"`
	MmDomainNameSimilarity float64            `ch:"mm_domain_name_similarity"`
	MmZip                  string             `ch:"mm_zip"`
	FinalUrl               string             `ch:"final_url"`
	Score                  float64            `ch:"score"`
	Tag                    string             `ch:"tag"`
	ResponseTimes          map[string]float32 `ch:"service_response_times"`
	ResponseCodes          map[string]uint16  `ch:"service_status_codes"`
	Timestamp              time.Time          `ch:"ts"`
}

func (r LlmTaskStoredResult) GetCreateQuery() string {
	query := fmt.Sprintf(
		`
        CREATE TABLE %s
        (
            duns String,
			is_active Bool,
			url String,
			verification_url String,
			status_code UInt16,
            error String,
            error_code String,
            error_type String,
            error_repr String,
			attempts_number Int32,
			crawler_errors Array(String),
			crawl_fails Array(String),
			crawled_pages Array(String),
			num_errors Int32,
			num_fails Int32,
			num_successes Int32,
			mm_name String,
			mm_address1 String,
			mm_address2 String,
			mm_city String,
			mm_state String,
			mm_country String,
			mm_domain_name_similarity Float32,
			mm_zip String,
			final_url String,
			score Float32,
			tag String,
            service_response_times Map(String, Float32),
            service_status_codes Map(String, UInt16),
			ts DateTime64(6, 'UTC'),
        )
        ENGINE = MergeTree
        ORDER BY (duns, url)
    `,
		config.C.Run.InsertionTableName,
	)
	return query
}
