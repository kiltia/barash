// Package config provides a way to configure the application.
package config

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
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
	API APIConfig `yaml:"api"      env:", prefix=API_"`
	// Settings related to the provider - the component that retrieves data from database
	Provider ProviderConfig `yaml:"provider" env:", prefix=PROVIDER_"`
	// Settings related to the fetcher - the component that fetches data from the API
	Fetcher FetcherConfig `yaml:"fetcher"  env:", prefix=FETCHER_"`
	// Settings related to the writer - the component that saves results to the database
	Writer WriterConfig `yaml:"writer"   env:", prefix=WRITER_"`
	// Logger configuration
	Log LogConfig `yaml:"log"      env:", prefix=LOG_"`
	// Graceful shutdown logic configuration
	Shutdown ShutdownConfig `yaml:"shutdown" env:", prefix=SHUTDOWN_"`

	// It can be two-table or continuous mode.
	// Two-table mode allows to get data from one table and save it to another.
	// It's expected that after draining all the data from the first table,
	// runner will be stopped.
	// Continuous mode allows to get data from the table and save it to the
	// same table.
	Mode RunnerMode `yaml:"mode" env:"RUN_MODE"`
}

type APIConfig struct {
	Type       string           `yaml:"type"           env:"TYPE"`
	RequestURL string           `yaml:"request_url"    env:"REQUEST_URL"`
	Method     RunnerHTTPMethod `yaml:"method"         env:"METHOD"`
	// Timeout
	APITimeout time.Duration `yaml:"api_timeout"    env:"TIMEOUT"`
	// Retries
	NumRetries  int           `yaml:"num_retries"    env:"N_RETRIES"`
	MinWaitTime time.Duration `yaml:"min_wait_time"  env:"MIN_WAIT_TIME"`
	MaxWaitTime time.Duration `yaml:"max_wait_time"  env:"MAX_WAIT_TIME"`
	// Request extension
	BodyFilePath string `yaml:"body_file_path" env:"BODY_FILE_PATH"`
}

type DatabaseCredentials struct {
	Username string `yaml:"username" env:"USER"`
	Password string `yaml:"password" env:"PASSWORD"`
}

type CircuitBreakerConfig struct {
	Enabled                 bool          `yaml:"enabled"                    env:"ENABLE"`
	MaxRequests             uint32        `yaml:"max_requests"               env:"MAX_REQUESTS"`
	ConsecutiveFailure      uint32        `yaml:"consecutive_failure"        env:"CONSECUTIVE_FAILURE"`
	TotalFailurePerInterval uint32        `yaml:"total_failure_per_interval" env:"TOTAL_FAILURE_PER_INTERVAL"`
	Interval                time.Duration `yaml:"interval"                   env:"INTERVAL"`
	Timeout                 time.Duration `yaml:"timeout"                    env:"TIMEOUT"`
}

type FetcherConfig struct {
	MinFetcherWorkers int `yaml:"min_fetcher_workers" env:"N_WORKERS"`
	MaxFetcherWorkers int `yaml:"max_fetcher_workers" env:"MAX_WORKERS"`
	// Warmup parameters
	Duration     time.Duration `yaml:"duration"            env:"WARMUP_TIME"`
	EnableWarmup bool          `yaml:"enable_warmup"       env:"ENABLE_WARMUP"`
	IdleTime     time.Duration `yaml:"idle_time"           env:"IDLE_TIME"`
	Timeout      time.Duration `yaml:"timeout"             env:"TIMEOUT"`

	// Circuit breaker can be configured to prevent Runner from overloading
	// the API or sending too much bad responses to Clickhouse.
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker" env:", prefix=CB_"`
}

type CorrectionConfig struct {
	EnableErrorsCorrection   bool          `yaml:"enable_errors_correction"`
	ErrorCorrection          time.Duration `yaml:"error_correction"`
	EnableTimeoutsCorrection bool          `yaml:"enable_timeouts_correction" env:"ENABLE_TIMEOUTS"`
	MaxTimeoutCorrection     time.Duration `yaml:"max_timeout_correction"     env:"TIMEOUTS"`
}

type ContinuousModeConfig struct {
	Freshness time.Duration `yaml:"freshness" env:"FRESHNESS"`
}

type ShutdownConfig struct {
	GracePeriod   time.Duration `yaml:"grace_period"    env:"GRACE_PERIOD"`
	DBSaveTimeout time.Duration `yaml:"db_save_timeout" env:"DB_SAVE_TIMEOUT"`
}

type SourceBackend = string

type DatabaseConfig struct {
	Backend string `yaml:"backend"`
	// Should be set with env vars
	Credentials DatabaseCredentials
	Host        string `yaml:"host"     env:"HOST"`
	Port        string `yaml:"port"     env:"PORT"`
	Database    string `yaml:"database" env:"DB"`
}

type SourceConfig struct {
	DatabaseConfig `       yaml:",inline"`
	SelectTable    string `yaml:"table"           env:"TABLE"`
	SelectSQLPath  string `yaml:"select_sql_path" env:"SELECT_SQL"`
}

type SinkConfig struct {
	DatabaseConfig `       yaml:",inline"`
	InsertTable    string `yaml:"table"   env:"TABLE"`
}

type ProviderConfig struct {
	SleepTime       time.Duration `yaml:"sleep_time"        env:"SLEEP_TIME"`
	SelectBatchSize int           `yaml:"select_batch_size" env:"SELECTION_BATCH_SIZE"`
	SelectRetries   int           `yaml:"select_retries"    env:"SELECT_RETRIES"`

	Source SourceConfig `yaml:"source"`

	// Continuous mode specific configuration
	ContinuousMode ContinuousModeConfig `yaml:"continuous_mode" env:", prefix=CONTINUOUS_"`
}

const (
	BackendClickhouse string = "clickhouse"
	BackendPostgres   string = "postgres"
)

type WriterConfig struct {
	InsertBatchSize int          `yaml:"insert_batch_size" env:"INSERT_BATCH_SIZE"`
	Sinks           []SinkConfig `yaml:"sinks"`
	SaveTag         string       `yaml:"save_tag"          env:"TAG"`

	// Runner uses corr_ts to generate "virtual" timestamp for the results.
	// This can be used to postpone, shuffle, retry new requests with the
	// same data. This is useful for the continuous mode.
	Correction CorrectionConfig `yaml:"correction" env:", prefix=CORRECTION_"`
}

type LogConfig struct {
	Level    zapcore.Level `yaml:"level"    env:"LEVEL"`
	Encoding string        `yaml:"encoding" env:"ENCODING"`
}

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "", "Path to YAML configuration file")
	_ = godotenv.Load() // load the user-defined `.env` file
}

func Load() (*Config, error) {
	flag.Parse()
	var cfg *Config
	var err error
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
		if configPath == "" {
			return nil, errors.New("config path is empty")
		}
	}
	// Load configuration
	if configPath == "" {
		// Load from environment variables only
		cfg = &Config{}
	} else {
		// Load from YAML file with environment variable overrides
		cfg, err = LoadFromYAML(configPath)
		if err != nil {
			log.Fatalf("loading configuration from %s: %v", configPath, err)
		}
	}
	return cfg, err
}

func LoadFromYAML(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
