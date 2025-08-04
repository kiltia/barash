package runner

import (
	"context"
	"sync"

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
	ctx context.Context,
	writerCh chan S,
	wg *sync.WaitGroup,
) {
	var batch []S

	saveBatch := func() {
		ctx, cancel := context.WithTimeout(
			context.Background(),
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

	done := make(chan struct{})

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	for {
		select {
		case <-ctx.Done():
			zap.S().Infow("Context is cancelled. Saving the remaining batch")
			saveBatch()
			zap.S().Infow("Batch is saved, writer is stopped")
			return
		case result, ok := <-writerCh:
			if !ok {
				zap.S().Infow("Channel is closed")
			}
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
		default:
			select {
			case <-done:
				zap.S().
					Infow("All workers are stopped. Saving the remaining batch")
				saveBatch()
				zap.S().Infow("Batch is saved, writer is stopped")
				return
			default:
				continue
			}
		}
	}
}
