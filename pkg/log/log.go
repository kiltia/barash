package log

import (
	"orb/runner/pkg/config"

	"go.uber.org/zap"
)

func Init(
	cfg config.LogConfig,
) {
	conf := zap.NewDevelopmentConfig()

	conf.Level = zap.NewAtomicLevelAt(cfg.Level)

	zap.ReplaceGlobals(zap.Must(conf.Build()))
}
