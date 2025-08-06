package runner

import (
	"context"

	"go.uber.org/zap"
)

// Writes a non-empty batch to the database.
func (r *Runner[S, R, P, Q]) write(
	ctx context.Context,
	batch []S,
) (err error) {
	zap.S().Debugw(
		"Saving processed batch to the database",
		"batch_len", len(batch),
	)
	err = r.clickHouseClient.InsertBatch(
		ctx,
		batch,
		r.cfg.Run.Tag,
	)
	if err != nil {
		zap.S().Errorw(
			"Failed to save processed batch to the database",
			"error", err,
		)
		return err
	}

	zap.S().Infow(
		"Saved processed batch to the database",
		"batch_len", len(batch),
	)
	return err
}

func (r *Runner[S, R, P, Q]) writer(
	resultsCh chan S,
) {
	var batch []S

	innerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	saveBatch := func() {
		ctx, cancel := context.WithTimeout(
			innerCtx,
			r.cfg.Timeouts.DBSaveTimeout,
		)
		defer cancel()
		err := r.write(ctx, batch)
		// TODO(nrydanov): Add reaction based on error returned
		// For example, if connection is dropped, we need to automatically
		// restore session
		// Source: https://github.com/kiltia/runner/issues/15
		if err == nil {
			batch = *new([]S)
		}
	}

	for result := range resultsCh {
		batch = append(
			batch,
			result,
		)
		if len(
			batch,
		) >= r.cfg.Run.InsertionBatchSize {
			zap.S().Infow(
				"Have enough results, saving to the database",
			)
			saveBatch()
		}
	}

	zap.S().
		Infow("All results processed by Runner, saving the rest to the database and exiting")
	saveBatch()
}
