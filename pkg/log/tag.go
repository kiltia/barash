package log

type LogTag uint

// Logger event tag.
const (
	LogTagUnset LogTag = iota
	LogTagUnknown

	LogTagClickHouse
	LogTagMain
	LogTagRunner
	LogTagStandby
	LogTagFetching
	LogTagWriting
	LogTagQualityControl
	LogTagLogParsing
	LogTagApiImpl
)

var tagToString = map[LogTag]string{
	LogTagUnknown:        "log_tag_unknown",
	LogTagUnset:          "log_tag_unset",
	LogTagClickHouse:     "clickhouse",
	LogTagRunner:         "runner",
	LogTagStandby:        "standby",
	LogTagFetching:       "fetching",
	LogTagWriting:        "writing",
	LogTagQualityControl: "quality_control",
	LogTagLogParsing:     "log_parsing",
	LogTagApiImpl:        "api_impl",
    LogTagMain: "main_loop",
}

// Implement [fmt.Stringer] interface.
func (e LogTag) String() string {
	if tag, ok := tagToString[e]; ok {
		return tag
	}
	return LogTagUnknown.String()
}
