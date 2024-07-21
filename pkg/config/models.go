package config

import (
	"go.uber.org/zap"
)

type Config struct {
	Api            ApiConfig            `yaml:"api"`
	ClickHouse     ClickHouseConfig     `yaml:"clickhouse"`
	Timeouts       TimeoutConfig        `yaml:"timeouts"`
	HttpRetries    RetryConfig          `yaml:"http_retries"`
	SelectRetries  RetryConfig          `yaml:"select_retries"`
	ZapLogger      zap.Config           `yaml:"zap_logger"`
	Run            RunConfig            `yaml:"run"`
	QualityControl QualityControlConfig `yaml:"quality_control_config"`
}

type QualityControlConfig struct {
	BatchTimeLimit   int     `yaml:"batch_time_limit"`
	SuccessThreshold float64 `yaml:"success_threshold"`
}

type ApiConfig struct {
	Name   string `yaml:"name"`
	Host   string `yaml:"host"`
	Port   string `yaml:"port"`
	Method string `yaml:"method"`
}

type ClickHouseConfig struct {
	Username string `yaml:"user"`
	Database string `yaml:"db"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
}

type TimeoutConfig struct {
	ApiTimeout       int `yaml:"api_timeout"`
	GoroutineTimeout int `yaml:"goroutine_timeout"`
}

type RetryConfig struct {
	NumRetries  int `yaml:"retries_number"`
	MinWaitTime int `yaml:"min_wait_time"`
	MaxWaitTime int `yaml:"max_wait_time"`
}

type RunConfig struct {
	FetcherWorkers int               `yaml:"fetcher_workers"`
	BatchSize      int               `yaml:"batch_size"`
	Freshness      int               `yaml:"freshness"`
	SleepTime      int               `yaml:"sleep_time"`
	Tag            string            `yaml:"tag"`
	ExtraParams    map[string]string `yaml:"extra_params"`
	Mode           RunnerMode        `yaml:"mode"`
}
