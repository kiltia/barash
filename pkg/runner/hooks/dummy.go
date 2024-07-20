package hooks

import (
	"context"

	rd "orb/runner/pkg/runner/data"
	ri "orb/runner/pkg/runner/interface"
)

type DummyHooks[S ri.StoredValue] struct{}

// Implement the [Hooks] interface.
func (dh *DummyHooks[S]) AfterBatch(
	ctx context.Context,
	batch rd.ProcessedBatch[S],
	qcFails *int,
) {
	// do nothing
}
