// Interfaces which a user of the runner needs to implement
// in order to verify their API.
package rinterface

import (
	"time"
)

type (
	StoredValue interface {
		// Return CREATE TABLE query for creating a new table
		// that will be used to store every run's data in a row.
		GetCreateQuery() string
	}

	StoredParams interface {
		GetUrl() string
	}

	Response[S StoredValue, P StoredParams] interface {
		IntoStored(
			params P,
			attemptNumber int,
			url string,
			body map[string]any,
			status int,
			timeElapsed time.Duration,
		) S
	}

	QueryBuilder[S StoredValue, P StoredParams] interface {
		GetSelectQuery() string

		// Updating [QueryBuilder]'s inner state
		// based on data that was selected from the database.
		UpdateState(
			batch []P,
		)

		// Refreshes inner state
		ResetState()
	}
)
