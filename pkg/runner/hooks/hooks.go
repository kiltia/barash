package hooks

import (
	"context"

	rd "orb/runner/pkg/runner/data"
	ri "orb/runner/pkg/runner/interface"
)

// NOTE(evgenymng): this interface's API is a subject to change
type Hooks[S ri.StoredValue] interface {
	AfterBatch(ctx context.Context, batch rd.ProcessedBatch[S], qcFails *int)
}
