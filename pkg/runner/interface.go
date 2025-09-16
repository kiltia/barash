package runner

import (
	"context"
	"encoding/json"
	"net/url"
	"time"
)

type (
	StoredResult interface {
		GetCreateQuery(tableName string) string
	}

	StoredParams any

	StoredParamsToQuery interface {
		GetQueryParams() url.Values
	}

	StoredParamsToBody interface {
		GetBody() []byte
	}

	Response[S StoredResult, P StoredParams] interface {
		IntoStored(
			request ServiceRequest[P],
			err error,
			attemptNumber int,
			status int,
			timeElapsed time.Duration,
			saveTag string,
		) S
	}

	// QueryBuilder interface represents an object which abstract SQL generation
	// logic.
	QueryBuilder[P StoredParams] interface {
		// FormatQuery formats SQL template using QueryBuilder's inner state.
		FormatQuery(sql string) string

		// UpdateState updates inner state based on batch data.
		UpdateState(
			batch []P,
		)

		// ResetState resets inner state
		ResetState()
	}

	// Source interface represents task storage.
	Source[P any] interface {
		GetNextBatch(
			ctx context.Context,
			sql string,
			qb QueryBuilder[P],
		) (result []P, err error)
	}

	// Sink interface represents result storage.
	Sink[S any] interface {
		InsertBatch(
			ctx context.Context,
			batch []S,
		) error
		InitTable(
			ctx context.Context,
		) error
	}

	// IncludeBodyFromFile interface is used to inject body to request
	IncludeBodyFromFile interface {
		SetBody(body json.RawMessage)
	}
)
