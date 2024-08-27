package meta

import (
	"fmt"
	"time"

	"orb/runner/pkg/config"
)

type VerifyResult struct {
	StatusCode     int
	TimeElapsed    time.Duration
	VerifyParams   VerifyParams
	RequestLink    string
	AttemptsNumber int
	Timestamp      time.Time
	MetaResponse   VerifyResponse
}

// Implement the [rinterface.StoredValue] interface.
func (r VerifyResult) GetInsertQuery() string {
	query := fmt.Sprintf(`
        INSERT INTO %s VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, now(), fromUnixTimestamp64Micro(?)
        )
    `, config.C.Run.InsertionTableName)
	return query
}

// Implement the [rinterface.StoredValue] interface.
func (r VerifyResult) GetStatusCode() int {
	return r.StatusCode
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
			status_code Int32,
			error String,
			fail String,
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
			ts DateTime,
			ts64 DateTime64(6,
			'UTC') DEFAULT fromUnixTimestamp64Micro(toUnixTimestamp64Micro(toDateTime64(ts,
			6,
			'UTC')) + toInt64(randUniform(1,
			1000000.)))
        )
        ENGINE = MergeTree
        ORDER BY (duns, url)
    `, config.C.Run.InsertionTableName)
	return query
}

// Implement the [rinterface.StoredValue] interface.
func (r VerifyResult) AsArray() []any {
	verifyParams := r.VerifyParams
	response := r.MetaResponse
	debugInfo := r.MetaResponse.DebugInfo
	pageStats := r.MetaResponse.DebugInfo.CrawlerDebug.PageStats
	crawlerDebug := debugInfo.CrawlerDebug
	MatchMaskSummary := response.MatchMask.MatchMaskSummary

	return []any{
		verifyParams.Duns,
		true,
		verifyParams.Url,
		r.RequestLink,
		r.StatusCode,
		response.Error,
		crawlerDebug.FailStatus,
		r.AttemptsNumber,
		crawlerDebug.CrawlerErrors,
		crawlerDebug.CrawlFails,
		crawlerDebug.CrawledPages,
		pageStats.Errors,
		pageStats.Fails,
		pageStats.Successes,
		debugInfo.Features,
		response.MatchMask.MatchMaskDetails,
		MatchMaskSummary.Name,
		MatchMaskSummary.Address1,
		MatchMaskSummary.Address2,
		MatchMaskSummary.City,
		MatchMaskSummary.State,
		MatchMaskSummary.Country,
		MatchMaskSummary.DomainNameSimilarity,
		response.FinalUrl,
		response.Score,
		config.C.Run.Tag,
		r.Timestamp.UnixMicro(),
	}
}
