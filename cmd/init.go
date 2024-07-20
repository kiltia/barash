package main

import (
	"fmt"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"

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

	log.S = zap.Must(config.C.ZapLogger.Build()).Sugar()
}
