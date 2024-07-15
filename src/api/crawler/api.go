package crawler

import (
	"context"

	rdata "orb/runner/src/runner/data"
)

type CrawlerApi struct{}

// Implement the [api.Api] interface.
func (srv *CrawlerApi) AfterBatch(
	ctx context.Context,
	batch rdata.ProcessedBatch[CrawlingResult],
	failCount *int,
) {
	// TODO(nrydanov): Add post-hook logic here if required
}
