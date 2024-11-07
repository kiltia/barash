package meta

import (
	"context"

	"orb/runner/pkg/log"
	"orb/runner/pkg/runner/hooks"
	"orb/runner/pkg/util"
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
	select {
	case <-ctx.Done():
		return
	default:
		successesWithScores := util.Reduce(
			util.Map(
				results,
				func(res VerifyResult) bool {
					return res.GetStatusCode() == 200 &&
						res.MetaResponse.Score != nil
				},
			),
			0,
			func(acc int, v bool) int {
				if v {
					return acc + 1
				}
				return acc
			},
		)
		log.S.Info(
			"Post-analyzed the processed batch",
			log.L().
				Add("successes_with_scores", successesWithScores),
		)
	}
}
