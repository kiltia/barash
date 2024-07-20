package log

import "go.uber.org/zap"

// Logger event tags.
const (
	TagClickHouseError   = "clickhouse_error"
	TagClickHouseSuccess = "clickhouse_success"

	TagResponseTimeout = "response_timeout"
	TagErrorResponse   = "error_response"
	TagFailResponse    = "fail_response"
	TagSuccessResponse = "success_response"
	TagRunnerDebug     = "runner_debug"
	TagRunnerStandby   = "runner_standby"

	TagQualityControl = "quality_control"
)

// The main logger instance used by this application.
var S *zap.SugaredLogger
