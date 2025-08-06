package crawler

// The query parameters, which are sent to the Crawler.
type CrawlerParams struct {
	URL              string    `query:"url"              ch:"url"`
	ID               int64     `query:"-"                ch:"id"`
	HeadlessStrategy string    `query:"headless_browser"`
	Fields           *[]string `query:"fields"`
}
