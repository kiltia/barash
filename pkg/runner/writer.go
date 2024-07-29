package runner

import (
	"context"
	"time"

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
	qcCh chan []S,
	nothingLeft chan bool,
) {
	logObject := log.L().Tag(log.LogTagWriting)
	var batch []S
	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-writerCh:
			if !ok {
				log.S.Info("Channel is closed", logObject)
			}
			batch = append(batch, result)
		case <-nothingLeft:
			log.S.Info(
				"Got \"nothing left\" signal, saving the rest of batch to database",
				logObject,
			)
			err := r.write(ctx, batch)
			if err != nil {
				log.S.Error(
					"Failed to save processed batch to the database",
					logObject.Error(err),
				)
			}
			qcCh <- batch
			batch = []S{}
			// TODO(nrydanov): Replace with config value
		case <-time.After(30 * time.Second):
			if len(batch) > config.C.Run.BatchSize {
				log.S.Info(
					"Have enough results, saving to the database", logObject,
				)
				err := r.write(ctx, batch)
				if err != nil {
					log.S.Error(
						"Failed to save processed batch to the database",
						logObject.Error(err),
					)
				}
				qcCh <- batch
				batch = []S{}
			}
		}
	}
}
