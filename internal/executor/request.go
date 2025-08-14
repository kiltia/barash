package executor

import (
	"encoding/json"

	"github.com/kiltia/runner/pkg/runner"
)

var _ runner.IncludeBodyFromFile = (*ExecutorParams)(nil)
var _ runner.StoredParams = (*ExecutorParams)(nil)
var _ runner.StoredParamsToBody = (*ExecutorParams)(nil)

type ExecutorParams struct {
	URL     string `query:"url"              ch:"url"`
	ID      int64  `query:"-"                ch:"id"`
	RawBody json.RawMessage
}

func (p *ExecutorParams) SetBody(body json.RawMessage) {
	p.RawBody = body
}

func (p *ExecutorParams) GetBody() []byte {
	return p.RawBody
}
