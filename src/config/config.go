package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var C *Config

func Load() (
	cfg Config,
	err error,
) {
	var content []byte
	filepath := fmt.Sprintf("configs/%s.yaml", getEnv("MOD", "dev"))
	content, err = os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("Got error: %v", err)
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
