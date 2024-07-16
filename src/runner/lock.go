package runner

import (
	"sync"

	"orb/runner/src/config"
	"orb/runner/src/log"
)

func lockUrl(url string, inProgress *sync.Map) bool {
	switch config.C.Api.Mode {
	case config.BatchMode:
		return true
	case config.ContiniousMode:
		_, loaded := inProgress.Load(url)
		if !loaded {
			inProgress.Store(url, true)
		}
		return !loaded
	default:
		log.S.Panicw("Unexpected batch mode", "input_value", config.C.Api.Mode)
	}

	// NOTE(nrydanov): Cringe...
	return false
}

func unlockUrl(url string, inProgress *sync.Map) {
	switch config.C.Api.Mode {
	case config.BatchMode:
		return
	case config.ContiniousMode:
		inProgress.Delete(url)
		return
	}
}
