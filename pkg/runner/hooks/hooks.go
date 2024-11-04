package hooks

import (
	"context"

	ri "orb/runner/pkg/runner/interface"
)

// NOTE(evgenymng): this interface's API is a subject to change
type Hooks[S ri.StoredValue] interface {
	AfterBatch(ctx context.Context, results []S)
}
