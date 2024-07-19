// Package containing runner's data structures.
package rdata

import (
	"time"

	ri "orb/runner/src/runner/interface"
)

type FetcherResult[S ri.StoredValue] struct {
	Value               S
	ProcessingStartTime time.Time
}

func NewFetcherResult[S ri.StoredValue](
	value S,
	processingStartTime time.Time,
) FetcherResult[S] {
	return FetcherResult[S]{
		Value:               value,
		ProcessingStartTime: processingStartTime,
	}
}

type ProcessedBatch[S ri.StoredValue] struct {
	Values         []S
	ProcessingTime time.Duration
}

func NewProcessedBatch[S ri.StoredValue](
	elements []S,
	processingTime time.Duration,
) ProcessedBatch[S] {
	return ProcessedBatch[S]{
		Values:         elements,
		ProcessingTime: processingTime,
	}
}

type QualityControlResult[S ri.StoredValue] struct {
	FailCount int
	Batch     ProcessedBatch[S]
}

func NewQualityControlResult[S ri.StoredValue](
	failCount int,
	batch ProcessedBatch[S],
) QualityControlResult[S] {
	return QualityControlResult[S]{
		FailCount: failCount,
		Batch:     batch,
	}
}
