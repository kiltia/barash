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
	Url              string           `json:"url"`
	HeadlessStrategy HeadlessStrategy `json:"headless_browser"`
	Fields           *[]string        `json:"field"`
}
