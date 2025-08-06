package main

import (
	"github.com/kiltia/runner/internal"
	"github.com/kiltia/runner/pkg/config"
)

func main() {
	var cfg config.Config
	config.Load(&cfg)

	internal.RunApplication(&cfg)
}
