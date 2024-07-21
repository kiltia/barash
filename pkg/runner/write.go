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
	err = r.clickHouseClient.AsyncInsertBatch(
		ctx,
		batch,
		config.C.Run.Tag,
	)
	if err != nil {
		log.S.Errorw(
			"Insertion to the ClickHouse database was unsuccessful!",
			"error", err,
			"tag", log.TagClickHouseError,
		)
		return err
	}

	log.S.Infow(
		"Insertion to the ClickHouse database was successful!",
		"batch_len", len(batch),
		"tag", log.TagClickHouseSuccess,
	)
	return err
}
