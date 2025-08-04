package config

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

const configFileEnvVar = "CONFIG_FILE"

func Init() {
	_ = godotenv.Load() // load the user-defined `.env` file
	// NOTE(evgenymng): godotenv.Load() does not override environment variables,
	// if they are already set. So, we first read the `.env` file and then
	// try to load the base configuration file.

	var baseEnvPath string
	if value, exists := os.LookupEnv(configFileEnvVar); exists {
		log.Printf(
			"Using the %s env variable to retrieve the config path\n",
			configFileEnvVar,
		)
		baseEnvPath = value
	} else {
		log.Printf("Base configuration file haven't been specified. "+
			"It will not be loaded. You can specify the path to the base configuration file "+
			"via the %s env variable or using the CLI arguments.\n", configFileEnvVar)
		return
	}

	if err := godotenv.Load(baseEnvPath); err != nil {
		log.Fatalf("Failed to read the base configuration file: %v", err)
	}
}

func Load(i *Config) {
	if err := envconfig.Process(context.Background(), i); err != nil {
		log.Fatal(err)
	}

	i.Run.ParsedExtraParams = parseExtraParams(i.Run.ExtraParams)
}

func parseExtraParams(extraParams string) map[string]string {
	if extraParams == "" {
		return nil
	}

	params := make(map[string]string)
	for param := range strings.SplitSeq(extraParams, ";") {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			continue
		}
		params[parts[0]] = parts[1]
	}
	return params
}
