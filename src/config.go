package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type RunnerConfig struct {
	VerifierConfig   VerifierConfig   `yaml:"verifier"`
	ClickHouseConfig ClickHouseConfig `yaml:"clickhouse"`
	Timeouts         Timeouts         `yaml:"timeouts"`
	Retries          Retries          `yaml:"retries"`
	RunConfig        RunConfig        `yaml:"run"`
}

type VerifierConfig struct {
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

type Timeouts struct {
	VerifierTimeout  int `yaml:"verifier_timeout"`
	GoroutineTimeout int `yaml:"goroutine_timeout"`
}

type Retries struct {
	NumRetries  int `yaml:"retries_number"`
	MinWaitTime int `yaml:"min_wait_time"`
	MaxWaitTime int `yaml:"max_wait_time"`
}

type RunConfig struct {
	ProducerWorkers int               `yaml:"producer_workers"`
	ConsumerWorkers int               `yaml:"consumer_workers"`
	BatchSize       int               `yaml:"batch_size"`
	ExtraParams     map[string]string `yaml:"extra_params"`
}

func NewRunnerConfig() *RunnerConfig {
	filepath := fmt.Sprintf("configs/%s.yaml", getEnv("MOD", "dev"))
	content, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("Gotten error: %v", err)
		return nil
	}
	var runnerConfig RunnerConfig
	yaml.Unmarshal(content, &runnerConfig)
	return &runnerConfig
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
