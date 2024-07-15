package crawler

import (
	"context"

	rdata "orb/runner/src/runner/data"
	"orb/runner/src/runner/hooks"
)

type CrawlerApiHooks struct {
	// NOTE(evgenymng): embed the dummy implementation just in case
	hooks.DummyHooks[CrawlingResult]
}

func (srv *CrawlerApiHooks) AfterBatch(
	ctx context.Context,
	batch rdata.ProcessedBatch[CrawlingResult],
	failCount *int,
) {
	return
}
