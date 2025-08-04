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
		GetBody() map[string]any
	}

	Response[S StoredResult, P StoredParams] interface {
		IntoStored(
			params P,
			attemptNumber int,
			url string,
			body map[string]any,
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
