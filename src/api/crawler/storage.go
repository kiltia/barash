package crawler

type CrawlingResult struct {
	StatusCode       int
	CrawlerParams    CrawlerParams
	RequestLink      string
	AttemptsNumber   int
	CrawlingResponse *CrawlerResponse
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlingResult) GetInsertQuery() string {
	return `
        INSERT INTO crawler VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now()
        )
    `
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlingResult) GetStatusCode() int {
	return r.StatusCode
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlingResult) GetSimpleSelectQuery() string {
	return `
        SELECT url from master LIMIT %d OFFSET %d
    `
}

func (r CrawlingResult) GetContiniousSelectQuery() string {
	return ""
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlingResult) GetCreateQuery() string {
	// TODO(evgenymng): Return something
	return ""
}

// Implement the [rinterface.StoredValue] interface.
func (r CrawlingResult) AsArray() []any {
	crawlingParams := r.CrawlerParams
	response := r.CrawlingResponse

	return []any{
		crawlingParams.Url,
		r.RequestLink,
		r.StatusCode,
		r.AttemptsNumber,
		response.OriginalUrl,
		response.FinalUrl,
		response.Status,
		response.ResponseSize,
		response.HeadlessUsed,
		response.Parsed.Urls,
	}
}
