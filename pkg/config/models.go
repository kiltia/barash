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
	API            APIConfig            `env:", prefix=API_"`
	ClickHouse     ClickHouseConfig     `env:", prefix=CLICKHOUSE_"`
	Log            LogConfig            `env:", prefix=LOG_"`
	CircuitBreaker CircuitBreakerConfig `env:", prefix=CB_"`
	Storage        StorageConfig        `env:", prefix=STORAGE_"`
	ContinuousMode ContinuousModeConfig `env:", prefix=CONTINUOUS_"`
	Shutdown       ShutdownConfig       `env:", prefix=SHUTDOWN_"`

	Provider   ProviderConfig   `env:", prefix=PROVIDER_"`
	Fetcher    FetcherConfig    `env:", prefix=FETCHER_"`
	Correction CorrectionConfig `env:", prefix=CORRECTION_"`

	Mode RunnerMode `env:"RUN_MODE"`
}

type APIConfig struct {
	// Connection data
	Name     string           `env:"NAME"`
	Host     string           `env:"HOST"`
	Port     string           `env:"PORT, default=80"`
	Endpoint string           `env:"ENDPOINT"`
	Method   RunnerHTTPMethod `env:"METHOD, default=GET"`
	// Timeout
	APITimeout time.Duration `env:"TIMEOUT, default=3m"`
	// Retries
	NumRetries        int               `env:"N_RETRIES, default=3"`
	MinWaitTime       time.Duration     `env:"MIN_WAIT_TIME, default=2s"`
	MaxWaitTime       time.Duration     `env:"MAX_WAIT_TIME, default=16s"`
	ExtraParams       string            `env:"EXTRA_PARAMS"`
	ParsedExtraParams map[string]string `                                     display:"-"`
}

type ClickHouseConfig struct {
	Database string `env:"DB"`
	Username string `env:"USER"`
	Password string `env:"PASSWORD"`
	Host     string `env:"HOST, default=127.0.0.1"`
	Port     string `env:"PORT, default=9000"`
}

type CircuitBreakerConfig struct {
	MaxRequests             uint32        `env:"MAX_REQUESTS, default=100"`
	ConsecutiveFailure      uint32        `env:"CONSECUTIVE_FAILURE, default=10"`
	TotalFailurePerInterval uint32        `env:"TOTAL_FAILURE_PER_INTERVAL, default=900"`
	Interval                time.Duration `env:"INTERVAL, default=60s"`
	Timeout                 time.Duration `env:"TIMEOUT, default=60s"`
}

type FetcherConfig struct {
	MinFetcherWorkers int `env:"START_FETCHER_WORKERS, default=400"`
	MaxFetcherWorkers int `env:"FETCHER_WORKERS, default=800"`
	// Warmup parameters
	Duration time.Duration `env:"WARMUP_TIME, default=60s"`
	Enabled  bool          `env:"ENABLE_WARMUP, default=false"`
	IdleTime time.Duration `env:"FETCHER_IDLE_TIME, default=10s"`
}

type CorrectionConfig struct {
	EnableErrorsCorrection   bool          `env:"ENABLE_ERRORS, default=false"`
	ErrorCorrection          time.Duration `env:"ERRORS, default=24h"`
	EnableTimeoutsCorrection bool          `env:"ENABLE_TIMEOUTS, default=true"`
	MaxTimeoutCorrection     time.Duration `env:"TIMEOUTS, default=504h"`
}

type StorageConfig struct {
	SelectionBatchSize int    `env:"SELECTION_BATCH_SIZE, default=40000"`
	InsertionBatchSize int    `env:"INSERTION_BATCH_SIZE, default=10000"`
	SelectionTableName string `env:"SELECTION_TABLE"`
	InsertionTableName string `env:"INSERTION_TABLE"`
	SelectRetries      int    `env:"SELECT_RETRIES, default=5"`
	Tag                string `env:"TAG"`
}

type ContinuousModeConfig struct {
	Freshness time.Duration `env:"FRESHNESS, default=168h"`
}

type ShutdownConfig struct {
	GracePeriod   time.Duration `env:"GRACE_PERIOD, default=60s"`
	DBSaveTimeout time.Duration `env:"DB_SAVE_TIMEOUT, default=30s"`
}

type ProviderConfig struct {
	SleepTime time.Duration `env:"SLEEP_TIME, default=1m"`
}

type LogConfig struct {
	Level    zapcore.Level `env:"LEVEL, default=warn"`
	Encoding string        `env:"ENCODING, default=console"`
}
