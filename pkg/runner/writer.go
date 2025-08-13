package runner

import (
	"context"

	"go.uber.org/zap"
)

func (r *Runner[S, R, P, Q]) writer(
	resultsCh chan S,
) {
	var batch []S

	innerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	saveBatch := func() {
		ctx, cancel := context.WithTimeout(
			innerCtx,
			r.cfg.Shutdown.DBSaveTimeout,
		)
		defer cancel()
		err := r.write(ctx, batch)
		// TODO(nrydanov): Add reaction based on error returned
		// For example, if connection is dropped, we need to automatically
		// restore session
		// Source: https://github.com/kiltia/runner/issues/15
		if err == nil {
			batch = *new([]S)
		} else {
			zap.S().Errorw(
				"saving processed batch to the database",
				"error", err,
			)
		}
	}

	for result := range resultsCh {
		batch = append(
			batch,
			result,
		)
		if len(batch) >= r.cfg.Writer.InsertionBatchSize {
			zap.S().Infow(
				"have enough results, saving to the database",
			)
			saveBatch()
		}
	}

	zap.S().
		Infow("all results processed, saving the rest to the database and exiting")
	saveBatch()
}

// Writes a non-empty batch to the database.
func (r *Runner[S, R, P, Q]) write(
	ctx context.Context,
	batch []S,
) (err error) {
	logger := zap.S().
		With("batch_len", len(batch))
	logger.Debugw(
		"saving processed batch to the database",
	)
	err = r.clickHouseClient.InsertBatch(
		ctx,
		batch,
		r.cfg.Writer.InsertTag,
	)
	if err != nil {
		return err
	}

	logger.Infow(
		"saved processed batch to the database",
	)
	return nil
}
