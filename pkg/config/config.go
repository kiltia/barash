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
		log.Fatalf("reading the base configuration file: %v", err)
	}
}

func Load(i *Config) {
	_ = godotenv.Load() // load the user-defined `.env` file
	if err := envconfig.Process(context.Background(), i); err != nil {
		log.Fatal(err)
	}

	i.API.ParsedExtraParams = parseExtraParams(i.API.ExtraParams)
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
