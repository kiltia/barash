// Interfaces which a user of the runner needs to implement
// in order to verify their API.
package rinterface

import "time"

type (
	StoredValue interface {
		// Return **parameterized** INSERT query for inserting a
		// row into the database.
		//
		// Note that, by design, the receiver value shouldn't
		// matter and it is possible for it to be zero-value. So don't
		// rely on it, or check before using it in your implementation.
		//
		// Ideally, you should only return a single parameterized
		// (templated) query.
		GetInsertQuery() string

		// Return CREATE TABLE query for creating a new table
		// that will be used to store every run's data in a row.
		GetCreateQuery() string

		// Return stored values as an array.
		//
		// The values are used as parameters to the INSERT query.
		AsArray() []any

		// Get value of the status code field.
		GetStatusCode() int
	}

	StoredRequest interface {
		GetUrl() string
	}

	Response[S StoredValue, P StoredRequest] interface {
		IntoStored(
			params P,
			attemptNumber int,
			url string,
			status int,
			timeElapsed time.Duration,
		) S
	}

	QueryBuilder[S StoredValue, P StoredRequest] interface {
		// Return SELECT query for retrieving a row from the database
		// in continious mode, which means that rows are retrieved
		// based on last processed time
		GetContiniousSelectQuery() string

		// Return SELECT query for retrieving rows from the database
		// in simple mode, which means that rows are retrieved
		// based on offset
		GetTwoTableSelectQuery() string

		// Updating QueryBuilder's inner state
		// based on data that was selected from the database
		UpdateState(batch []P)

		// Refreshes inner state
		ResetState()
	}
)
