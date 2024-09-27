package runner

import (
	"context"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
)

// Writes a non-empty batch to the database.
func (r *Runner[S, R, P, Q]) write(
	ctx context.Context,
	batch []S,
) (err error) {
	logObject := log.L().Tag(log.LogTagWriting)

	log.S.Debug("Saving processed batch to the database", logObject)
	err = r.clickHouseClient.AsyncInsertBatch(
		ctx,
		batch,
		config.C.Run.Tag,
	)
	if err != nil {
		log.S.Error(
			"Failed to save processed batch to the database",
			logObject.Error(err),
		)
		return err
	}

	log.S.Info(
		"Saved processed batch to the database",
		logObject.Add("batch_len", len(batch)),
	)
	return err
}

func (r *Runner[S, R, P, Q]) writer(
	ctx context.Context,
	writerCh chan S,
	nothingLeft chan bool,
) {
	logObject := log.L().Tag(log.LogTagWriting)
	var batch []S

	saveBatch := func() {
		err := r.write(ctx, batch)
		if err != nil {
			log.S.Error(
				"Failed to save processed batch to the database",
				logObject.Error(err),
			)
		}
		batch = *new([]S)
	}

	for {
		select {
		case <-ctx.Done():
			saveBatch()
			return
		case result, ok := <-writerCh:
			if !ok {
				log.S.Info("Channel is closed", logObject)
			}
			batch = append(batch, result)
			if len(batch) >= config.C.Run.InsertionBatchSize {
				log.S.Info(
					"Have enough results, saving to the database", logObject,
				)
				saveBatch()
			}
		case <-nothingLeft:
			log.S.Info(
				"Got \"nothing left\" signal, saving the rest of batch to database",
				logObject,
			)
			saveBatch()
		}
	}
}
