package config

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type RunnerConfig struct {
	ApiConfig            ApiConfig            `yaml:"verifier"`
	ClickHouseConfig     ClickHouseConfig     `yaml:"clickhouse"`
	Timeouts             TimeoutConfig        `yaml:"timeouts"`
	HttpRetries          RetryConfig          `yaml:"http_retries"`
	SelectRetries        RetryConfig          `yaml:"select_retries"`
	LoggerConfig         zap.Config           `yaml:"logger"`
	RunConfig            RunConfig            `yaml:"run"`
	QualityControlConfig QualityControlConfig `yaml:"quality_control_config"`
}

type QualityControlConfig struct {
	Period    int     `yaml:"period"`
	Threshold float64 `yaml:"threshold"`
}

type ApiConfig struct {
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
	VerifierTimeout  int `yaml:"verifier_timeout"`
	GoroutineTimeout int `yaml:"goroutine_timeout"`
}

type RetryConfig struct {
	NumRetries  int `yaml:"retries_number"`
	MinWaitTime int `yaml:"min_wait_time"`
	MaxWaitTime int `yaml:"max_wait_time"`
}

type RunConfig struct {
	ProducerWorkers       int               `yaml:"producer_workers"`
	ConsumerWorkers       int               `yaml:"consumer_workers"`
	SelectionBatchSize    int               `yaml:"selection_batch_size"`
	VerificationBatchSize int               `yaml:"verification_batch_size"`
	InsertionBatchSize    int               `yaml:"insertion_batch_size"`
	DayOffset             int               `yaml:"day_offset"`
	SleepTime             int               `yaml:"sleep_time"`
	Tag                   string            `yaml:"tag"`
	ExtraParams           map[string]string `yaml:"extra_params"`
}

func Load() (
	runnerConfig RunnerConfig,
	err error,
) {
	var content []byte
	filepath := fmt.Sprintf("configs/%s.yaml", getEnv("MOD", "dev"))
	content, err = os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("Got error: %v", err)
		return runnerConfig, err
	}
	err = yaml.Unmarshal(content, &runnerConfig)
	return runnerConfig, err
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
