package metaapi

import (
	"context"

	"orb/runner/src/log"
	rdata "orb/runner/src/runner/data"
	"orb/runner/src/runner/util"
)

type MetaApi struct{}

// Implement the [api.Api] interface.
func (srv *MetaApi) AfterBatch(
	ctx context.Context,
	batch rdata.ProcessedBatch[VerificationResult],
	failCount *int,
) {
	select {
	case <-ctx.Done():
		return
	default:
		successesWithScores := util.Reduce(
			util.Map(batch.Values, func(res VerificationResult) bool {
				return res.GetStatusCode() == 200 &&
					res.VerificationResponse.Score != nil
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
