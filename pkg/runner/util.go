package runner

import (
	"context"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
)

func (r *Runner[S, R, P, Q]) standby(
	ctx context.Context,
) error {
	logObject := log.L().Tag(log.LogTagStandby)

	waitTime := time.Duration(config.C.Run.SleepTime) * time.Second
	log.S.Info("The runner is entering standby mode", logObject)
	defer log.S.Info("The runner has left standby mode", logObject)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		return nil
	}
}
