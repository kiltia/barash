package meta

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
)

var MasterColumnNames = [30]string{
	"duns",
	"is_active",
	"url",
	"verification_url",
	"status_code",
	"error",
	"error_type",
	"error_code",
	"error_repr",
	"attempts_number",
	"crawler_errors",
	"crawl_fails",
	"crawled_pages",
	"num_errors",
	"num_fails",
	"num_successes",
	"features",
	"match_mask_details",
	"mm_name",
	"mm_address1",
	"mm_address2",
	"mm_city",
	"mm_state",
	"mm_country",
	"mm_domain_name_similarity",
	"final_url",
	"score",
	"tag",
	"ts",
	"corr_ts",
}

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
	values := make(
		[]string,
		len(MasterColumnNames),
	)
	for i := range values {
		values[i] = "?"
	}
	query := fmt.Sprintf(
		`
        INSERT INTO %s (%s) VALUES (%s)
    `,
		config.C.Run.InsertionTableName,
		strings.Join(
			MasterColumnNames[:],
			", ",
		),
		strings.Join(
			values,
			", ",
		),
	)

	log.S.Debug(
		"Formed insert query: ",
		log.L().
			Add("query", query),
	)
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

// Implement the [rinterface.StoredValue] interface.
func (r VerifyResult) AsDict() map[string]any {
	verifyParams := r.VerifyParams
	response := r.MetaResponse
	debugInfo := r.MetaResponse.DebugInfo
	pageStats := r.MetaResponse.DebugInfo.CrawlerDebug.PageStats
	crawlerDebug := debugInfo.CrawlerDebug
	MatchMaskSummary := response.MatchMask.MatchMaskSummary

	var score float64
	if response.Score == nil {
		score = math.NaN()
	} else {
		score = *response.Score
	}

	var correctedTs time.Time

	// NOTE(nrydanov): Need to replace with certain error code when we'll
	// determine it.
	if response.Error.Code != nil &&
		strings.Contains(strings.ToLower(*response.Error.Code), "timeout") {
		log.S.Debug("Timeout was detected, timestamp will be corrected", log.L())
		// NOTE(nrydanov): This is a hack to avoid sitations when
		// too many potential timeouts are present in batch.
		seconds := rand.Intn(24 * 60 * 60 * config.C.Run.MaxCorrection)
		correctedTs = r.Timestamp.Add(
			time.Duration(
				seconds,
			) * time.Second,
		)
	} else {
		correctedTs = r.Timestamp
	}

	return map[string]any{
		"duns":                      verifyParams.Duns,
		"is_active":                 true,
		"url":                       verifyParams.Url,
		"verification_url":          r.RequestLink,
		"status_code":               r.StatusCode,
		"error":                     response.Error.Reason,
		"error_type":                response.Error.ErrorType,
		"error_code":                response.Error.Code,
		"error_repr":                response.Error.ErrorRepr,
		"attempts_number":           r.AttemptsNumber,
		"crawler_errors":            crawlerDebug.CrawlerErrors,
		"crawl_fails":               crawlerDebug.CrawlFails,
		"crawled_pages":             crawlerDebug.CrawledPages,
		"num_errors":                pageStats.Errors,
		"num_fails":                 pageStats.Fails,
		"num_successes":             pageStats.Successes,
		"features":                  debugInfo.FeatureExtractorDebug.Features,
		"match_mask_details":        response.MatchMask.MatchMaskDetails,
		"mm_name":                   MatchMaskSummary.Name,
		"mm_address1":               MatchMaskSummary.Address1,
		"mm_address2":               MatchMaskSummary.Address2,
		"mm_city":                   MatchMaskSummary.City,
		"mm_state":                  MatchMaskSummary.State,
		"mm_country":                MatchMaskSummary.Country,
		"mm_domain_name_similarity": MatchMaskSummary.DomainNameSimilarity,
		"final_url":                 response.FinalUrl,
		"score":                     score,
		"tag":                       config.C.Run.Tag,
		"ts":                        r.Timestamp,
		"corr_ts":                   correctedTs,
	}
}
