package main

// import (
// 	"fmt"

// 	"go.uber.org/zap"
// )

// type StorageProvider[Stored StoredData] interface {

// 	// Inserts one batch with verification results to database
// 	AsyncInsertBatch(
// 		batch []Stored,
// 		tag string,
// 	) error

// 	// Retrieves input for next batch processing
// 	// SelectNextBatch(days int, selectBatchSize int) (*[]RequestData, error)
// }

// func GetStorageProvider[S any](config RunnerConfig, logger *zap.SugaredLogger) (StorageProvider[I, P], error) {
// 	switch backend := config.StorageBackend; backend {
// 	}
// 	ch, version, err := NewClickHouseClient(config.ClickHouseConfig)
// 	if err != nil {
// 		logger.Errorw(
// 			"Connection to the ClickHouse database was unsuccessful!",
// 			"error", err,
// 			"tag", CLICKHOUSE_ERROR_TAG,
// 		)
// 		return nil, err
// 	} else {
// 		logger.Infow(
// 			"Connection to the ClickHouse database was successful!",
// 			"tag", CLICKHOUSE_SUCCESS_TAG,
// 		)
// 		logger.Infow(
// 			fmt.Sprintf("%v", version),
// 			"tag", CLICKHOUSE_SUCCESS_TAG,
// 		)
// 	}

// 	return ch, nil
// }
