package config

import (
	"time"

	"go.uber.org/zap/zapcore"
)

type RunnerMode string

const (
	TwoTableMode   RunnerMode = "two-table"
	ContinuousMode RunnerMode = "continuous"
)

type RunnerHTTPMethod string

const (
	RunnerHTTPMethodGet  RunnerHTTPMethod = "GET"
	RunnerHTTPMethodPost RunnerHTTPMethod = "POST"
)

type Config struct {
	API           APIConfig         `env:", prefix=API_"`
	ClickHouse    ClickHouseConfig  `env:", prefix=CLICKHOUSE_"`
	Timeouts      TimeoutConfig     `env:", prefix=TIMEOUTS_"`
	HTTPRetries   HTTPRetryConfig   `env:", prefix=HTTP_RETRIES_"`
	SelectRetries SelectRetryConfig `env:", prefix=SELECT_RETRIES_"`
	Log           LogConfig         `env:", prefix=LOG_"`
	Run           RunConfig         `env:", prefix=RUN_"`
}

type APIConfig struct {
	Name     string           `env:"NAME"`
	Host     string           `env:"HOST"`
	Port     string           `env:"PORT, default=80"`
	Endpoint string           `env:"ENDPOINT"`
	Method   RunnerHTTPMethod `env:"METHOD, default=GET"`
}

type ClickHouseConfig struct {
	Database string `env:"DB"`
	Username string `env:"USER"`
	Password string `env:"PASSWORD"`
	Host     string `env:"HOST, default=127.0.0.1"`
	Port     string `env:"PORT, default=9000"`
}

type TimeoutConfig struct {
	APITimeout      time.Duration `env:"API_TIMEOUT, default=3m"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT, default=10s"`
	DBSaveTimeout   time.Duration `env:"DB_SAVE_TIMEOUT, default=30s"`
}

type HTTPRetryConfig struct {
	NumRetries  int           `env:"NUMBER, default=3"`
	MinWaitTime time.Duration `env:"MIN_WAIT_TIME, default=2s"`
	MaxWaitTime time.Duration `env:"MAX_WAIT_TIME, default=16s"`
}

type SelectRetryConfig struct {
	NumRetries int `env:"NUMBER, default=5"`
}

type RunConfig struct {
	MaxFetcherWorkers         int               `env:"MAX_FETCHER_WORKERS, default=800"`
	MinFetcherWorkers         int               `env:"MIN_FETCHER_WORKERS, default=400"`
	SelectionBatchSize        int               `env:"SELECTION_BATCH_SIZE, default=40000"`
	InsertionBatchSize        int               `env:"INSERTION_BATCH_SIZE, default=10000"`
	SelectionTableName        string            `env:"SELECTION_TABLE"`
	InsertionTableName        string            `env:"INSERTION_TABLE"`
	MaxCorrection             time.Duration     `env:"MAX_CORRECTION, default=504h"`
	ServerErrorCorrectionDiff time.Duration     `env:"SERVER_ERROR_CORRECTION_DIFF, default=24h"`
	Freshness                 time.Duration     `env:"FRESHNESS, default=168h"`
	SleepTime                 time.Duration     `env:"SLEEP_TIME, default=1m"`
	WarmupTime                time.Duration     `env:"WARMUP_TIME, default=3m"`
	FetcherIdleTime           time.Duration     `env:"FETCHER_IDLE_TIME, default=10s"`
	Tag                       string            `env:"TAG"`
	ExtraParams               string            `env:"EXTRA_PARAMS"`
	ParsedExtraParams         map[string]string `                                                display:"-"`
	Mode                      RunnerMode        `env:"MODE"`
}

type LogConfig struct {
	Level    zapcore.Level `env:"LEVEL, default=warn"`
	Encoding string        `env:"ENCODING, default=console"`
}
