package crawler

import (
	"context"

	rdata "orb/runner/src/runner/data"
	"orb/runner/src/runner/hooks"
)

type CrawlerApiHooks struct {
	// NOTE(evgenymng): embed the dummy implementation just in case
	hooks.DummyHooks[CrawlerResult]
}

func (srv *CrawlerApiHooks) AfterBatch(
	ctx context.Context,
	batch rdata.ProcessedBatch[CrawlerResult],
	failCount *int,
) {
}
