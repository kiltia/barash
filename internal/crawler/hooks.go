package crawler

import (
	"context"

	rdata "orb/runner/pkg/runner/data"
	"orb/runner/pkg/runner/hooks"
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
