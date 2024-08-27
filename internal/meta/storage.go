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
	panic("Method is not implemented")
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
