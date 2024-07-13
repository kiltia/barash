// Package containing runner's data structures.
package rdata

import (
	"time"

	ri "orb/runner/src/runner/interface"
)

type FetcherResult[S ri.StoredValueType] struct {
	Value               S
	ProcessingStartTime time.Time
}

func NewFetcherResult[S ri.StoredValueType](
	value S,
	processingStartTime time.Time,
) FetcherResult[S] {
	return FetcherResult[S]{
		Value:               value,
		ProcessingStartTime: processingStartTime,
	}
}

type ProcessedBatch[S ri.StoredValueType] struct {
	Values              []S
	ProcessingStartTime time.Time
}

func NewProcessedBatch[S ri.StoredValueType](
	elements []S,
	processingStartTime time.Time,
) ProcessedBatch[S] {
	return ProcessedBatch[S]{
		Values:              elements,
		ProcessingStartTime: processingStartTime,
	}
}

type QualityControlResult[S ri.StoredValueType] struct {
	FailCount int
	Batch     ProcessedBatch[S]
}
