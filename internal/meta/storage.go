package meta

import (
	"fmt"
	"time"

	"orb/runner/pkg/config"
)

type VerifyResult struct {
	Duns                   string    `ch:"duns"`
	IsActive               bool      `ch:"is_active"`
	Url                    string    `ch:"url"`
	VerificationUrl        string    `ch:"verification_url"`
	StatusCode             uint16    `ch:"status_code"`
	Error                  string    `ch:"error"`
	ErrorCode              string    `ch:"error_code"`
	ErrorType              string    `ch:"error_type"`
	ErrorRepr              string    `ch:"error_repr"`
	AttemptsNumber         uint8     `ch:"attempts_number"`
	CrawlerErrors          []string  `ch:"crawler_errors"`
	CrawlFails             []string  `ch:"crawl_fails"`
	CrawledPages           []string  `ch:"crawled_pages"`
	NumErrors              int       `ch:"num_errors"`
	NumFails               int       `ch:"num_fails"`
	NumSuccesses           int       `ch:"num_successes"`
	Features               string    `ch:"features"`
	MatchMaskDetails       string    `ch:"match_mask_details"`
	MmName                 string    `ch:"mm_name"`
	MmAddress1             string    `ch:"mm_address1"`
	MmAddress2             string    `ch:"mm_address2"`
	MmCity                 string    `ch:"mm_city"`
	MmState                string    `ch:"mm_state"`
	MmCountry              string    `ch:"mm_country"`
	MmDomainNameSimilarity float64   `ch:"mm_domain_name_similarity"`
	FinalUrl               string    `ch:"final_url"`
	Score                  float64   `ch:"score"`
	Tag                    string    `ch:"tag"`
	ResponseTimes          map[string]float32 `ch:"response_times"`
	ResponseCodes          map[string]uint16  `ch:"response_codes"`
	FeStatusCode           []uint16    `ch:"metrics.fe_status_code"`
	FeResponseTime         []float32   `ch:"metrics.fe_response_time"`
	FtStatusCode           []uint16    `ch:"metrics.ft_status_code"`
	FtResponseTime         []float32   `ch:"metrics.ft_response_time"`
	ScorerStatusCode       []uint16    `ch:"metrics.scorer_status_code"`
	ScorerResponseTime     []float32   `ch:"metrics.scorer_response_time"`
	Timestamp              time.Time `ch:"timestamp"`
	Ts                     time.Time `ch:"ts"`
	CorrTs                 time.Time `ch:"corr_ts"`
}

// Implement the [rinterface.StoredValue] interface.
func (r VerifyResult) GetCreateQuery() string {
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
			features String,
			match_mask_details String,
			mm_name String,
			mm_address1 String,
			mm_address2 String,
			mm_city String,
			mm_state String,
			mm_country String,
			mm_domain_name_similarity Float32,
			final_url String,
			score Float32,
			tag String,
            response_times Map(String, float32),
            service_status_codes Map(String, uint16)
			ts DateTime64(6, 'UTC'),
			corr_ts DateTime64(6, 'UTC')
        )
        ENGINE = MergeTree
        ORDER BY (duns, url)
    `,
		config.C.Run.InsertionTableName,
	)
	return query
}
