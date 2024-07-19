package meta

type MetaResult struct {
	StatusCode     int
	MetaRequest    MetaRequest
	RequestLink    string
	AttemptsNumber int
	MetaResponse   MetaResponse
}

// Implement the [rinterface.StoredValue] interface.
func (r MetaResult) GetInsertQuery() string {
	return `
        INSERT INTO master VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, now()
        )
    `
}

// Implement the [rinterface.StoredValue] interface.
func (r MetaResult) GetStatusCode() int {
	return r.StatusCode
}

// Implement the [rinterface.StoredValue] interface.
func (r MetaResult) GetCreateQuery() string {
	panic("Method is not implemented")
}

// Implement the [rinterface.StoredValue] interface.
func (r MetaResult) AsArray() []any {
	verifyParams := r.MetaRequest
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
	}
}
