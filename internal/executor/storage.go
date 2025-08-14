package executor

import (
	"encoding/json"
	"fmt"
	"time"
)

type ExecutorResult struct {
	URL            string          `ch:"url"`
	RequestLink    string          `ch:"request_link"`
	StatusCode     int             `ch:"status_code"`
	Error          string          `ch:"error"`
	ErrorType      string          `ch:"error_type"`
	ErrorCode      string          `ch:"error_code"`
	AttemptsNumber uint8           `ch:"attempts_number"`
	Urls           []string        `ch:"urls"`
	TaskResult     json.RawMessage `ch:"task_result"`
	TimeElapsed    float64         `ch:"time_elapsed"`
	Tag            string          `ch:"tag"`
	Timestamp      time.Time       `ch:"ts"`
}

// Implement the [rinterface.StoredValue] interface.
func (r ExecutorResult) GetCreateQuery(tableName string) string {
	query := fmt.Sprintf(
		`
        CREATE TABLE %s
        (
            url String,
            request_link String,
			status_code UInt8,
            error String,
            error_type String,
            error_code String,
            attempts_number UInt8,
            urls Array(String),
			task_result String,
			time_elapsed Float64,
            tag String,
            ts DateTime64
        )
        ENGINE = MergeTree
        ORDER BY ts
    `, tableName)
	return query
}
