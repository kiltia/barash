package config

import (
	"time"

	"go.uber.org/zap"
)

type RunnerMode string

const (
	TwoTableMode   RunnerMode = "two-table"
	ContinuousMode RunnerMode = "continuous"
)

type RunnerHttpMethod string

const (
	GET  RunnerHttpMethod = "GET"
	POST RunnerHttpMethod = "POST"
)

type Config struct {
	Api           ApiConfig         `env:", prefix=API_"`
	ClickHouse    ClickHouseConfig  `env:", prefix=CLICKHOUSE_"`
	Timeouts      TimeoutConfig     `env:", prefix=TIMEOUTS_"`
	HttpRetries   HttpRetryConfig   `env:", prefix=HTTP_RETRIES_"`
	SelectRetries SelectRetryConfig `env:", prefix=SELECT_RETRIES_"`
	Log           LogConfig         `env:", prefix=LOG_"`
	Run           RunConfig         `env:", prefix=RUN_"`
}

type ApiConfig struct {
	Name     string           `env:"NAME, required"`
	Host     string           `env:"HOST, required"`
	Port     string           `env:"PORT, default=80"`
	Endpoint string           `env:"ENDPOINT, required"`
	Method   RunnerHttpMethod `env:"METHOD, default=GET"`
}

type ClickHouseConfig struct {
	Database string `env:"DB, required"`
	Username string `env:"USER, required"`
	Password string `env:"PASSWORD, required"`
	Host     string `env:"HOST, deafult=127.0.0.1"`
	Port     string `env:"PORT, default=9000"`
}

type TimeoutConfig struct {
	ApiTimeout      time.Duration `env:"API_TIMEOUT, default=3m"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT, default=10s"`
	DbSaveTimeout   time.Duration `env:"DB_SAVE_TIMEOUT, default=10s"`
}

type HttpRetryConfig struct {
	NumRetries  int           `env:"NUMBER, default=3"`
	MinWaitTime time.Duration `env:"MIN_WAIT_TIME, default=2s"`
	MaxWaitTime time.Duration `env:"MAX_WAIT_TIME, default=16s"`
}

type SelectRetryConfig struct {
	NumRetries int `env:"NUMBER, default=5"`
}

type RunConfig struct {
	MaxFetcherWorkers  int               `env:"MAX_FETCHER_WORKERS, default=800"`
	MinFetcherWorkers  int               `env:"MIN_FETCHER_WORKERS, default=400"`
	SelectionBatchSize int               `env:"SELECTION_BATCH_SIZE, default=40000"`
	InsertionBatchSize int               `env:"INSERTION_BATCH_SIZE, default=10000"`
	SelectionTableName string            `env:"SELECTION_TABLE, required"`
	InsertionTableName string            `env:"INSERTION_TABLE"`
	MaxCorrection      int               `env:"MAX_CORRECTION, default=21"` // days
	Freshness          int               `env:"FRESHNESS, default=7"`       // days
	SleepTime          time.Duration     `env:"SLEEP_TIME, default=1m"`
	WarmupTime         time.Duration     `env:"WARMUP_TIME, default=3m"`
	Tag                string            `env:"TAG"`
	ExtraParams        map[string]string `env:"EXTRA_PARAMS"`
	Mode               RunnerMode        `env:"MODE, required"`
}

type LogConfig struct {
	Level    zap.AtomicLevel `yaml:"LEVEL, default=warn"`
	Encoding string          `yaml:"ENCODING, default=console"`
}
