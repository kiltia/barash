package crawler

import "orb/runner/pkg/runner/hooks"

type CrawlerApiHooks struct {
	// NOTE(evgenymng): embed the dummy implementation just in case
	hooks.DummyHooks[CrawlerResult]
}
