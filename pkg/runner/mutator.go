package runner

import (
	"encoding/json"
	"os"

	"github.com/kiltia/runner/pkg/config"
)

type IncludeBodyFromFile interface {
	SetBody(body json.RawMessage)
}

type BodyMutator[P IncludeBodyFromFile] struct {
	body json.RawMessage
}

func NewBodyMutator[P IncludeBodyFromFile](
	cfg *config.Config,
) BodyMutator[P] {
	filePath := cfg.API.BodyFilePath
	// read the file

	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	return BodyMutator[P]{
		body: json.RawMessage(file),
	}
}

func (m *BodyMutator[P]) Mutate(params P) {
	params.SetBody(m.body)
}
