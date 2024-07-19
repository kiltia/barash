package runner

import (
	"context"
	"time"

	"orb/runner/src/config"
	"orb/runner/src/log"
	rd "orb/runner/src/runner/data"
	rr "orb/runner/src/runner/request"
	"orb/runner/src/runner/util"
)

func (r *Runner[S, R, P, Q]) fetcher(
	ctx context.Context,
	fetcherNum int,
	tasks chan *rr.GetRequest[P],
	results chan rd.FetcherResult[S],
	nothingLeft chan bool,
) {
	for {
		select {
		case task, ok := <-tasks:
			startTime := time.Now()

			if !ok {
				// TODO(nrydanov): What should we do when channels are closed?
				return
			}
			// NOTE(nrydanov): If task is nil, then it was the last signal
			// sent to channel from main thread. We need to ask for more
			// tasks
			if task == nil {
				log.S.Infow(
					"Fetcher has no work left, asking for a new batch",
					"fetcher_num", fetcherNum,
					"tag", log.TagRunnerDebug,
				)
				nothingLeft <- true
				break
			}

			log.S.Debugw(
				"Sending request to get page contents",
				"fetcher_num",
				fetcherNum,
			)
			resultList, err := r.SendGetRequest(ctx, *task)
			if err != nil {
				log.S.Errorw(
					"There was an error, while sending the request "+
						"to the subject API",
					"error", err,
					"fetcher_num", fetcherNum,
				)
			}

			log.S.Debugw("Sending fetching results", "fetcher_num", fetcherNum)
			for _, result := range resultList {
				results <- rd.NewFetcherResult(result, startTime)
			}
		case <-ctx.Done():
			log.S.Debugw(
				"Fetcher's context is cancelled",
				"fetcher_num", fetcherNum,
				"error", ctx.Err(),
			)
		}
	}
}

func (r *Runner[S, R, P, Q]) writer(
	ctx context.Context,
	writerNum int,
	results chan rd.FetcherResult[S],
	processedBatches chan rd.ProcessedBatch[S],
	initialTime time.Time,
) {
	var batch []S
	lastBatchTime := initialTime
	insertBatch := func() {
		if len(batch) == 0 {
			log.S.Infow("Batch is empty, SQL insert query won't be executed", "writer_num", writerNum)
			return
		}
		err := r.clickHouseClient.AsyncInsertBatch(
			ctx,
			batch,
			config.C.Run.Tag,
		)
		if err != nil {
			log.S.Errorw(
				"Insertion to the ClickHouse database was unsuccessful!",
				"error",
				err,
				"writer_num",
				writerNum,
				"tag",
				log.TagClickHouseError,
			)
			return
		}
		log.S.Infow(
			"Insertion to the ClickHouse database was successful!",
			"batch_len", len(batch),
			"writer_num", writerNum,
			"tag", log.TagClickHouseSuccess,
		)

		processedBatches <- rd.NewProcessedBatch(batch, time.Since(lastBatchTime))
		batch = []S{}
		lastBatchTime = time.Now()
	}

	for {
		select {
		case result, ok := <-results:
			if !ok {
				// TODO(nrydanov): What should we do if channel is closed?
				return
			}

			batch = append(batch, result.Value)

			// NOTE(nrydanov): Task count is determined at the moment
			// of selecting next batch. We use this variable to determine
			// whether all fetchers have done it's tasks or not
			if len(batch) >= config.C.Run.BatchSize {
				insertBatch()
			}

		case <-ctx.Done():
			log.S.Debugw(
				"Writer's context is cancelled",
				"writer_num", writerNum,
				"error", ctx.Err(),
				"tag", log.TagRunnerDebug,
			)
			return
		}
	}
}

func (r *Runner[S, R, P, Q]) qualityControl(
	ctx context.Context,
	processedBatches chan rd.ProcessedBatch[S],
	qcResults chan rd.QualityControlResult[S],
) {
	for {
		select {
		case batch, ok := <-processedBatches:
			if !ok {
				return
			}

			failCount := 0
			numRequests := len(batch.Values)
			numSuccesses := util.Reduce(
				util.Map(batch.Values, func(res S) int {
					return res.GetStatusCode()
				}),
				0,
				func(acc int, v int) int {
					if v == 200 {
						return acc + 1
					}
					return acc
				},
			)

			// NOTE(nrydanov): Case 1. Batch processing takes too much time
			if batch.ProcessingTime > time.Duration(
				config.C.QualityControl.BatchTimeLimit,
			)*time.Second {
				log.S.Infow(
					"Batch processing takes longer than it should.",
					"num_successes", numSuccesses,
					"num_requests", numRequests,
					"tag", log.TagQualityControl,
				)
				failCount++
			}

			// NOTE(nrydanov): Case 2. Too many requests ends with errors
			if numSuccesses < int(
				float64(numRequests)*config.C.QualityControl.SuccessThreshold,
			) {
				log.S.Infow(
					"Too many 5xx errors from the API.",
					"tag", log.TagQualityControl,
				)
				failCount++
			}
			qcResults <- rd.NewQualityControlResult(failCount, batch)

		case <-ctx.Done():
			log.S.Debugw(
				"Quality control routine's context is cancelled",
				"error", ctx.Err(),
			)
			return
		}
	}
}
