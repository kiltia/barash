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
