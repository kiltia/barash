package hooks

import (
	"context"

	ri "orb/runner/pkg/runner/interface"
)

type DummyHooks[S ri.StoredValue] struct{}

// Implement the [Hooks] interface.
func (dh *DummyHooks[S]) AfterBatch(
	ctx context.Context,
	results []S,
) {
	// do nothing
}
