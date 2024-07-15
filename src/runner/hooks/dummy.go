package hooks

import (
	"context"

	rd "orb/runner/src/runner/data"
	ri "orb/runner/src/runner/interface"
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
