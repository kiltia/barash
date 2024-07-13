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

func (r *Runner[S, R, P]) fetcher(
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
				log.S.Debugw(
					"There was an error, while sending the request "+
						"to the subject API",
					"error", err,
				)
				break
			}

			for _, result := range resultList {
				results <- rd.NewFetcherResult(result, startTime)
			}

		case <-ctx.Done():
			log.S.Debugw(
				"Fetcher's context is cancelled",
				"fetcher_num", fetcherNum,
				"error", ctx.Err(),
			)
			return
		}
	}
}

func (r *Runner[S, R, P]) writer(
	ctx context.Context,
	consumerNum int,
	results chan rd.FetcherResult[S],
	processedBatches chan rd.ProcessedBatch[S],
) {
	var oldest *time.Time
	var batch []S
	for {
		select {
		case result, ok := <-results:
			if !ok {
				return
			}

			batch = append(batch, result.Value)
			if oldest == nil || result.ProcessingStartTime.Before(*oldest) {
				oldest = &result.ProcessingStartTime
			}

			if len(batch) >= config.C.Run.InsertionBatchSize {
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
						"consumer_num",
						consumerNum,
						"tag",
						log.TagClickHouseError,
					)
					break
				}
				log.S.Infow(
					"Insertion to the ClickHouse database was successful!",
					"batch_len", len(batch),
					"consumer_num", consumerNum,
					"tag", log.TagClickHouseSuccess,
				)

				processedBatches <- rd.NewProcessedBatch(batch, *oldest)
				batch = []S{}
				oldest = nil
			}

		case <-ctx.Done():
			log.S.Debugw(
				"Consumer's context is cancelled",
				"consumer_num", consumerNum,
				"error", ctx.Err(),
				"tag", log.TagRunnerDebug,
			)
			return
		}
	}
}

func (r *Runner[S, R, P]) qualityControl(
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
			sinceBatchStart := time.Since(batch.ProcessingStartTime)

			// NOTE(nrydanov): Case 1. Batch processing takes too much time
			if sinceBatchStart > time.Duration(
				r.qualityControlConfig.Period,
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
				float64(numRequests)*r.qualityControlConfig.Threshold,
			) {
				log.S.Infow(
					"Too many 5xx errors from the API.",
					"tag", log.TagQualityControl,
				)
				failCount++
			}
			qcResults <- rd.QualityControlResult[S]{
				FailCount: failCount,
				Batch:     batch,
			}

		case <-ctx.Done():
			log.S.Debugw(
				"Quality control routine's context is cancelled",
				"error", ctx.Err(),
			)
			return
		}
	}
}
