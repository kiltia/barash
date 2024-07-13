// Interfaces which a user of the runner needs to implement
// in order to verify their API.
package rinterface

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

		// Return SELECT query for retrieving a row from the database.
		//
		// TODO(evgenymng): rework the query building flow,
		// because now we expect from a user that they will
		// have a specific number of template parameters in that
		// query, which describe how old the records should be
		// (which makes no sense in general).
		GetSelectQuery() string

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

	Response[S StoredValue, P StoredParams] interface {
		IntoStored(params P, attemptNumber int, url string, status int) S
	}

	StoredParams interface{}
)
