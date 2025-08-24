package config

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

const configFileEnvVar = "CONFIG_FILE"

func Init() {
	_ = godotenv.Load() // load the user-defined `.env` file
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
	if err := envconfig.Process(context.Background(), i); err != nil {
		log.Fatal(err)
	}
}
