package api

import (
	"context"

	rd "orb/runner/src/runner/data"
	ri "orb/runner/src/runner/interface"
)

type Api[S ri.StoredValueType] interface {
	AfterBatch(context.Context, rd.ProcessedBatch[S])
}
