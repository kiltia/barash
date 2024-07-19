package meta

import (
	"context"

	"orb/runner/src/log"
	rdata "orb/runner/src/runner/data"
	"orb/runner/src/runner/hooks"
	"orb/runner/src/runner/util"
)

type VerifyApiHooks struct {
	// NOTE(evgenymng): embed the dummy implementation just in case
	hooks.DummyHooks[VerifyResult]
}

// Implement the [hooks.Hooks] interface.
func (srv *VerifyApiHooks) AfterBatch(
	ctx context.Context,
	batch rdata.ProcessedBatch[VerifyResult],
	failCount *int,
) {
	select {
	case <-ctx.Done():
		return
	default:
		successesWithScores := util.Reduce(
			util.Map(batch.Values, func(res VerifyResult) bool {
				return res.GetStatusCode() == 200 &&
					res.MetaResponse.Score != nil
			}),
			0,
			func(acc int, v bool) int {
				if v {
					return acc + 1
				}
				return acc
			},
		)
		log.S.Infow(
			"Post-analyzed the processed batch",
			"successes_with_scores", successesWithScores,
		)
	}
}
