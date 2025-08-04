package log

import (
	"orb/runner/pkg/config"

	"go.uber.org/zap"
)

var S Logger

type Logger struct {
	internal *zap.SugaredLogger
}

func (l Logger) GetInternal() *zap.SugaredLogger {
	return l.internal
}

func Init(
	cfg config.LogConfig,
) {
	conf := zap.NewDevelopmentConfig()

	conf.Level = zap.NewAtomicLevelAt(cfg.Level)

	zap.ReplaceGlobals(zap.Must(conf.Build()))
}
