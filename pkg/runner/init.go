package runner

import (
	"context"

	"github.com/kiltia/runner/pkg/config"

	"go.uber.org/zap"
)

func (r *Runner[S, R, P, Q]) initTable(
	ctx context.Context,
) {
	if r.cfg.Run.Mode == config.ContinuousMode {
		zap.S().
			Infow("running in continuous mode, skipping table initialization")
		return
	}
	var nilInstance S
	err := r.clickHouseClient.Connection.Exec(
		ctx,
		nilInstance.GetCreateQuery(r.cfg.Run.InsertionTableName),
	)

	if err != nil {
		zap.S().Warnw("table creation script has failed", "error", err)
	} else {
		zap.S().Infow("successfully initialized table for the Runner results")
	}
}
