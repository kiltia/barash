package crawler

type HeadlessStrategy int8

const (
	HeadlessStrategyFalse HeadlessStrategy = iota
	HeadlessStrategyTrue
	HeadlessStrategySmart
	HeadlessStrategyRandom
)

// The query parameters, which are sent to the Crawler.
type CrawlerRequest struct {
	Url              string    `json:"url"              ch:"url"`
	HeadlessStrategy string    `json:"headless_browser"`
	Fields           *[]string `json:"fields"`
}
