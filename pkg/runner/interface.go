package runner

import (
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
			tag string,
		) S
	}

	QueryBuilder[S StoredResult, P StoredParams] interface {
		GetSelectQuery() string

		UpdateState(
			batch []P,
		)

		ResetState()
	}
)
