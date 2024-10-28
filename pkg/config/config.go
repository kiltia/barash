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
		if value, exists := os.LookupEnv("CONFIG_FILE"); exists {
			filepath = value
		} else {
			return cfg, fmt.Errorf("Please, provide a configuration file name either via " +
				"CONFIG_FILE env variable or using the CLI arguments")
		}
	}
	content, err = os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("Failed to read the config file: %v", err)
		return cfg, err
	}
	err = yaml.Unmarshal(content, &cfg)
	return cfg, err
}

func LoadEnv(cfg Config) Config {
	if value, exists := os.LookupEnv("RUN_SELECTION_TABLE"); exists {
		cfg.Run.SelectionTableName = value
	}
	if value, exists := os.LookupEnv("RUN_INSERTION_TABLE"); exists {
		cfg.Run.InsertionTableName = value
	}
	if value, exists := os.LookupEnv("CLICKHOUSE_HOST"); exists {
		cfg.ClickHouse.Host = value
	}
	if value, exists := os.LookupEnv("CLICKHOUSE_PORT"); exists {
		cfg.ClickHouse.Port = value
	}
	if value, exists := os.LookupEnv("CLICKHOUSE_DB"); exists {
		cfg.ClickHouse.Database = value
	}
	if value, exists := os.LookupEnv("CLICKHOUSE_USER"); exists {
		cfg.ClickHouse.Username = value
	}
	if value, exists := os.LookupEnv("CLICKHOUSE_PASSWORD"); exists {
		cfg.ClickHouse.Password = value
	}
	return cfg
}
