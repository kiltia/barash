package hooks

import (
	"context"

	rd "orb/runner/src/runner/data"
	ri "orb/runner/src/runner/interface"
)

// NOTE(evgenymng): this interface's API is a subject to change
type Hooks[S ri.StoredValue] interface {
	AfterBatch(ctx context.Context, batch rd.ProcessedBatch[S], qcFails *int)
}
