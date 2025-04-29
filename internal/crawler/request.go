package crawler

// The query parameters, which are sent to the Crawler.
type CrawlerParams struct {
	Url              string    `query:"url"              ch:"url"`
	Id               int64     `query:"-"                ch:"id"`
	HeadlessStrategy string    `query:"headless_browser"`
	Fields           *[]string `query:"fields"`
}
