package metaapi

type VerificationResult struct {
	StatusCode           int
	VerifyParams         VerifyRequestParams
	VerificationLink     string
	AttemptsNumber       int
	VerificationResponse *VerifyResponse
}

// Implement the [rinterface.StoredValue] interface.
func (r VerificationResult) GetInsertQuery() string {
	return `
        INSERT INTO master VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, now()
        )
    `
}

// Implement the [rinterface.StoredValue] interface.
func (r VerificationResult) GetStatusCode() int {
	return r.StatusCode
}

// Implement the [rinterface.StoredValue] interface.
func (r VerificationResult) GetSelectQuery() string {
	return `
        with last as (
            select duns, url, max(ts) as max_ts
            from wv.master
            where is_active = True
            group by duns, url
        ),
        batch as (
            select duns, url, max_ts
            from last
            where max_ts < (now() - toIntervalDay(%d))
            limit %d
        ),
        final as (
            select
                batch.duns as duns,
                batch.url as url,
                gdmi.name,
                gdmi.loc_address1, gdmi.loc_address2,
                gdmi.loc_city, gdmi.loc_state,
                gdmi.loc_zip, gdmi.loc_country,
                gdmi.mail_address1, gdmi.mail_address2,
                gdmi.mail_city, gdmi.mail_state,
                gdmi.mail_zip, gdmi.mail_country
            from wv.gdmi_compact gdmi
            inner join batch using (duns)
            where gdmi.duns != '' and batch.url != ''
            order by cityHash64(batch.duns, batch.url)
        )
        select * from final
    `
}

// Implement the [rinterface.StoredValue] interface.
func (r VerificationResult) GetCreateQuery() string {
	// TODO(evgenymng): Return something
	return ""
}

// Implement the [rinterface.StoredValue] interface.
func (r VerificationResult) AsArray() []any {
	verifyParams := r.VerifyParams
	response := r.VerificationResponse
	debugInfo := r.VerificationResponse.DebugInfo
	pageStats := r.VerificationResponse.DebugInfo.CrawlerDebug.PageStats
	crawlerDebug := debugInfo.CrawlerDebug
	MatchMaskSummary := response.MatchMask.MatchMaskSummary

	return []any{
		verifyParams.Duns,
		true,
		verifyParams.Url,
		r.VerificationLink,
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
