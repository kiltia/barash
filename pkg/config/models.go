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
	// Configuration of interaction between the runner and the API
	API APIConfig `env:", prefix=API_"`
	// Logger configuration
	Log LogConfig `env:", prefix=LOG_"`
	// Circuit breaker can be configured to prevent Runner from overloading
	// the API or sending too much bad responses to Clickhouse.
	CircuitBreaker CircuitBreakerConfig `env:", prefix=CB_"`
	// Continuous mode specific configuration
	ContinuousMode ContinuousModeConfig `env:", prefix=CONTINUOUS_"`
	// Graceful shutdown logic configuration
	Shutdown ShutdownConfig `env:", prefix=SHUTDOWN_"`
	// Settings related to the provider - the component that retrieves data
	// from database
	Provider ProviderConfig `env:", prefix=PROVIDER_"`
	// Settings related to the fetcher - the component that fetches data
	// from the API
	Fetcher FetcherConfig `env:", prefix=FETCHER_"`
	// Settings related to the writer - the component that saves results to
	// the database
	Writer WriterConfig `env:", prefix=WRITER_"`
	// Runner uses corr_ts to generate "virtual" timestamp for the results.
	// This can be used to postpone, shuffle, retry new requests with the
	// same data. This is useful for the continuous mode.
	Correction CorrectionConfig `env:", prefix=CORRECTION_"`

	// It can be two-table or continuous mode.
	// Two-table mode allows to get data from one table and save it to another.
	// It's expected that after draining all the data from the first table,
	// runner will be stopped.
	// Continuous mode allows to get data from the table and save it to the
	// same table.
	Mode RunnerMode `env:"RUN_MODE"`
}

type APIConfig struct {
	// Connection data
	Type     string           `env:"TYPE"`
	Host     string           `env:"HOST"`
	Port     string           `env:"PORT, default=80"`
	Scheme   string           `env:"SCHEME, default=http"`
	Endpoint string           `env:"ENDPOINT"`
	Method   RunnerHTTPMethod `env:"METHOD, default=GET"`
	// Timeout
	APITimeout time.Duration `env:"TIMEOUT, default=3m"`
	// Retries
	NumRetries  int               `env:"N_RETRIES, default=3"`
	MinWaitTime time.Duration     `env:"MIN_WAIT_TIME, default=2s"`
	MaxWaitTime time.Duration     `env:"MAX_WAIT_TIME, default=16s"`
	ExtraParams map[string]string `env:"EXTRA_PARAMS"`

	BodyFilePath string `env:"BODY_FILE_PATH"`
}

type DatabaseConfig struct {
	Database string `env:"DB"`
	Username string `env:"USER"`
	Password string `env:"PASSWORD"`
	Host     string `env:"HOST, default=127.0.0.1"`
	Port     string `env:"PORT, default=9000"`
}

type CircuitBreakerConfig struct {
	Enabled                 bool          `env:"ENABLE, default=false"`
	MaxRequests             uint32        `env:"MAX_REQUESTS, default=100"`
	ConsecutiveFailure      uint32        `env:"CONSECUTIVE_FAILURE, default=10"`
	TotalFailurePerInterval uint32        `env:"TOTAL_FAILURE_PER_INTERVAL, default=900"`
	Interval                time.Duration `env:"INTERVAL, default=60s"`
	Timeout                 time.Duration `env:"TIMEOUT, default=60s"`
}

type FetcherConfig struct {
	MinFetcherWorkers int `env:"N_WORKERS, default=400"`
	MaxFetcherWorkers int `env:"MAX_WORKERS, default=800"`
	// Warmup parameters
	Duration     time.Duration `env:"WARMUP_TIME, default=60s"`
	EnableWarmup bool          `env:"ENABLE_WARMUP, default=false"`
	IdleTime     time.Duration `env:"IDLE_TIME, default=10s"`
	Timeout      time.Duration `env:"TIMEOUT, default=40s"`
}

type CorrectionConfig struct {
	EnableErrorsCorrection   bool          `env:"ENABLE_ERRORS, default=false"`
	ErrorCorrection          time.Duration `env:"ERRORS, default=24h"`
	EnableTimeoutsCorrection bool          `env:"ENABLE_TIMEOUTS, default=true"`
	MaxTimeoutCorrection     time.Duration `env:"TIMEOUTS, default=504h"`
}

type ContinuousModeConfig struct {
	Freshness time.Duration `env:"FRESHNESS, default=168h"`
}

type ShutdownConfig struct {
	GracePeriod   time.Duration `env:"GRACE_PERIOD, default=60s"`
	DBSaveTimeout time.Duration `env:"DB_SAVE_TIMEOUT, default=30s"`
}

type SourceBackend = string

const (
	SourceBackendClickhouse SourceBackend = "ch"
	SourceBackendPostgres   SourceBackend = "pg"
)

type SourceConfig struct {
	Backend     SourceBackend  `env:"BACKEND, default=ch"`
	Credentials DatabaseConfig `env:", prefix=CREDENTIALS"`
}

type ProviderConfig struct {
	SleepTime          time.Duration `env:"SLEEP_TIME, default=1m"`
	SelectionBatchSize int           `env:"SELECTION_BATCH_SIZE, default=40000"`
	SelectionTableName string        `env:"SELECTION_TABLE"`
	SelectRetries      int           `env:"SELECT_RETRIES, default=5"`
	SelectSQLPath      string        `env:"SELECT_SQL, default=select.sql"`

	Source SourceConfig `env:", prefix=SOURCE"`
}

type SinkBackend = string

const (
	SinkBackendClickhouse SinkBackend = "ch"
	SinkBackendPostgres   SinkBackend = "postgres"
)

type SinkConfig struct {
	Backend     SinkBackend    `env:"BACKEND, default=ch"`
	Credentials DatabaseConfig `env:", prefix=CREDENTIALS"`
}

type WriterConfig struct {
	InsertionBatchSize int        `env:"INSERT_BATCH_SIZE, default=10000"`
	InsertionTableName string     `env:"INSERT_TABLE"`
	Sink               SinkConfig `env:"SINK"`
	InsertSQLPath      string     `env:"INSERT_SQL, default=insert.sql"`
	SaveTag            string     `env:"TAG"`
}

type LogConfig struct {
	Level    zapcore.Level `env:"LEVEL, default=debug"`
	Encoding string        `env:"ENCODING, default=console"`
}
