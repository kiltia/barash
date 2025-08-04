package main

import (
	"orb/runner/internal"
	"orb/runner/pkg/config"
)

func main() {
	var cfg config.Config
	config.Load(&cfg)

	internal.RunApplication(&cfg)
}
