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
	var filepath string
	if len(os.Args) > 2 {
		api := os.Args[1]
		mode := os.Args[2]
		fmt.Printf("Using CLI settings to retrieve config path\n")
		filepath = fmt.Sprintf("config/%s.%s.yml", api, mode)
	} else {
		fmt.Printf("Using environment settings to retrieve config path\n")
		filepath = getEnv("CONFIG_FILE", "config/meta.dev.yml")
	}
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
