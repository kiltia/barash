package barash

import (
	"encoding/json"
	"os"
)

type BodyMutator struct {
	body json.RawMessage
}

func NewBodyMutator(
	bodyPath string,
) BodyMutator {
	filePath := bodyPath
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
