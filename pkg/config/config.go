package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type (
	RunnerMode string
)

const (
	TwoTableMode   RunnerMode = "two-table"
	ContinuousMode RunnerMode = "continuous"
)

var C *Config

func Load() (
	cfg Config,
	err error,
) {
	var content []byte
	filepath := getEnv("CONFIG_FILE", "configs/meta.dev.yml")
	content, err = os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("Failed to read the config file: %v", err)
		return cfg, err
	}
	err = yaml.Unmarshal(content, &cfg)
	return cfg, err
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
