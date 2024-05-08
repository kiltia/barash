package main

import (
	"os"
	"strconv"
)

type RunnerConfig struct {
	VerifierCreds    VerifierConfig
	ClickHouseConfig ClickHouseConfig
	VerifierTimeout  int
	GoroutineTimeout int
}

func NewRunnerConfig() *RunnerConfig {
	verifierTimeout, err := strconv.Atoi(getEnv("VERIFIER_TIMEOUT", "300"))
	if err != nil {
		return nil
	}
	goroutineTimeout, err := strconv.Atoi(getEnv("GOROUTINE_TIMEOUT", "300"))
	if err != nil {
		return nil
	}
	return &RunnerConfig{
		VerifierCreds:    *NewVerifierConfig(),
		ClickHouseConfig: *NewClickHouseConfig(),
		VerifierTimeout:  verifierTimeout,
		GoroutineTimeout: goroutineTimeout,
	}
}

type VerifierConfig struct {
	Host   string
	Port   string
	Method string
}

func NewVerifierConfig() *VerifierConfig {
	return &VerifierConfig{
		Host:   getEnv("VERIFIER_HOST", "127.0.0.1"),
		Port:   getEnv("VERIFIER_PORT", "8081"),
		Method: getEnv("VERIFIER_METHOD", "/verify"),
	}
}

type ClickHouseConfig struct {
	Username string
	Database string
	Password string
	Host     string
	Port     string
}

func NewClickHouseConfig() *ClickHouseConfig {
	return &ClickHouseConfig{
		Username: getEnv("CLICKHOUSE_USER", "user"),
		Database: getEnv("CLICKHOUSE_DB", "orb"),
		Password: getEnv("CLICKHOUSE_PASSWORD", "12345"),
		Host:     getEnv("CLICKHOUSE_HOST", "127.0.0.1"),
		Port:     getEnv("CLICKHOUSE_PORT", "9000"),
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}
