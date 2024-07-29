package config

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Api            apiConfig            `yaml:"api"`
	ClickHouse     clickHouseConfig     `yaml:"clickhouse"`
	Timeouts       timeoutConfig        `yaml:"timeouts"`
	HttpRetries    retryConfig          `yaml:"http_retries"`
	SelectRetries  retryConfig          `yaml:"select_retries"`
	Log            logConfig            `yaml:"log"`
	Run            runConfig            `yaml:"run"`
	QualityControl qualityControlConfig `yaml:"quality_control_config"`
}

type qualityControlConfig struct {
	BatchTimeLimit   int     `yaml:"batch_time_limit"`
	SuccessThreshold float64 `yaml:"success_threshold"`
}

type apiConfig struct {
	Name   string `yaml:"name"`
	Host   string `yaml:"host"`
	Port   string `yaml:"port"`
	Method string `yaml:"method"`
}

type clickHouseConfig struct {
	Username string `yaml:"user"`
	Database string `yaml:"db"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
}

type timeoutConfig struct {
	ApiTimeout       int `yaml:"api_timeout"`
	GoroutineTimeout int `yaml:"goroutine_timeout"`
}

type retryConfig struct {
	NumRetries  int `yaml:"retries_number"`
	MinWaitTime int `yaml:"min_wait_time"`
	MaxWaitTime int `yaml:"max_wait_time"`
}

type runConfig struct {
	MaxFetcherWorkers int               `yaml:"max_fetcher_workers"`
	MinFetcherWorkers int               `yaml:"min_fetcher_workers"`
	BatchSize         int               `yaml:"batch_size"`
	Freshness         int               `yaml:"freshness"`
	SleepTime         int               `yaml:"sleep_time"`
	HeatTime          int               `yaml:"heat_time"`
	Tag               string            `yaml:"tag"`
	ExtraParams       map[string]string `yaml:"extra_params"`
	Mode              RunnerMode        `yaml:"mode"`
}

type logConfig struct {
	Level            zap.AtomicLevel `yaml:"level"`
	Encoding         string          `yaml:"encoding"`
	OutputPaths      []string        `yaml:"output_paths"`
	ErrorOutputPaths []string        `yaml:"error_output_paths"`
	DevMode          bool            `yaml:"dev_mode"`
	EncoderConfig    encoderConfig   `yaml:"encoder_config"`
}

type encoderConfig struct {
	MessageKey    string               `yaml:"message_key"`
	LevelKey      string               `yaml:"level_key"`
	LevelEncoder  zapcore.LevelEncoder `yaml:"level_encoder"`
	TimeKey       string               `yaml:"time_key"`
	TimeEncoder   zapcore.TimeEncoder  `yaml:"time_encoder"`
	NameKey       string               `yaml:"name_key"`
	CallerKey     string               `yaml:"caller_key"`
	FunctionKey   string               `yaml:"function_key"`
	StacktraceKey string               `yaml:"stacktrace_key"`
}
