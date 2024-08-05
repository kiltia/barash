package crawler

type HeadlessStrategy int8

const (
	HeadlessStrategyFalse HeadlessStrategy = iota
	HeadlessStrategyTrue
	HeadlessStrategySmart
	HeadlessStrategyRandom
)

// The query parameters, which are sent to the Crawler.
type CrawlerParams struct {
	Url              string    `json:"url"              ch:"url"`
	Id               int64     `json:"-"                ch:"id"`
	HeadlessStrategy string    `json:"headless_browser"`
	Fields           *[]string `json:"fields"`
}

func (p CrawlerParams) GetUrl() string {
	return p.Url
}
