package meta

import (
	"context"

	"orb/runner/pkg/runner/hooks"
)

type VerifyApiHooks struct {
	// NOTE(evgenymng): embed the dummy implementation just in case
	hooks.DummyHooks[VerifyResult]
}

// Implement the [hooks.Hooks] interface.
func (srv *VerifyApiHooks) AfterBatch(
	ctx context.Context,
	results []VerifyResult,
) {
}
