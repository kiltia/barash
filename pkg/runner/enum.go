package runner

type ContextKey int

const (
	ContextKeyUnsuccessfulResponses ContextKey = iota
	ContextKeyFetcherNum
)
