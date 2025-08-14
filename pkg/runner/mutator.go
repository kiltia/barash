package runner

import (
	"encoding/json"
	"os"

	"github.com/kiltia/runner/pkg/config"
)

type IncludeBodyFromFile interface {
	SetBody(body json.RawMessage)
}

type BodyMutator struct {
	body json.RawMessage
}

func NewBodyMutator(
	cfg *config.Config,
) BodyMutator {
	filePath := cfg.API.BodyFilePath
	// read the file

	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	return BodyMutator{
		body: json.RawMessage(file),
	}
}

func (m *BodyMutator) Mutate(obj IncludeBodyFromFile) {
	obj.SetBody(m.body)
}
