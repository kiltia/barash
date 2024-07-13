package api

import (
	"context"

	rd "orb/runner/src/runner/data"
	ri "orb/runner/src/runner/interface"
)

type Api[S ri.StoredValue] interface {
	AfterBatch(ctx context.Context, batch rd.ProcessedBatch[S], qcFails *int)
}
