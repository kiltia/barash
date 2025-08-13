package main

import (
	"github.com/kiltia/runner/internal"
	"github.com/kiltia/runner/pkg/config"
)

func main() {
	var cfg config.Config
	// load enviroment variables + .env file, then base config file
	config.Init()
	// save variables from .env + base config file
	config.Load(&cfg)

	internal.RunApplication(&cfg)
}
