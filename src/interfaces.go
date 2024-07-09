package main

type StoredValueType interface {
	// Return **parameterized** INSERT query for inserting a
	// row into the database.
	//
	// Note that, by convention, the receiver value shouldn't
	// matter and it is possible for it to be [nil]. So don't
	// rely on it, or check before using it in your implementation.
	//
	// Ideally, you should only return a single parameterized
	// (templated) query.
	GetInsertQuery() string

	// Return SELECT query for retrieving a row from the database.
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

	// AdditionalSuccessLogic()
}

type ResponseType[S StoredValueType, P ParamsType] interface {
	IntoWith(params P, attemptNumber int, url string, status int) S
}

type ParamsType interface{}
