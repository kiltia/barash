package main

import (
	"fmt"

	"orb/runner/src/config"
	"orb/runner/src/log"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Print("No .env file found. Continuing...\n")
	}

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	config.C = &cfg

	log.S = zap.Must(config.C.Logger.Build()).Sugar()
}
