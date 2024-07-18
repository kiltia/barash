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
	Logger         zap.Config           `yaml:"logger"`
	Run            RunConfig            `yaml:"run"`
	QualityControl QualityControlConfig `yaml:"quality_control_config"`
}

type QualityControlConfig struct {
	Period    int     `yaml:"period"`
	Threshold float64 `yaml:"threshold"`
}

type ApiConfig struct {
	Type   ApiType `yaml:"type"`
	Host   string  `yaml:"host"`
	Port   string  `yaml:"port"`
	Method string  `yaml:"method"`
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

type TableData struct {
	ColumnType string `yaml:"type"`
	ColumnName string `yaml:"column"`
	Freshness  int    `yaml:"freshness"`
}

type RunConfig struct {
	FetcherWorkers   int               `yaml:"fetcher_workers"`
	WriterWorkers    int               `yaml:"writer_workers"`
	RequestBatchSize int               `yaml:"request_batch_size"`
	BatchSize        int               `yaml:"batch_size"`
	SleepTime        int               `yaml:"sleep_time"`
	Tag              string            `yaml:"tag"`
	ExtraParams      map[string]string `yaml:"extra_params"`
	Mode             RunnerMode        `yaml:"mode"`
	TableData        TableData         `yaml:"table_data"`
}
