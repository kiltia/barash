package executor

import (
	"fmt"

	"github.com/kiltia/runner/pkg/config"
)

type ExecutorQueryBuilder struct {
	LastID             int64
	BatchSize          int
	Mode               config.RunnerMode
	SelectionTableName string
}

func (qb *ExecutorQueryBuilder) UpdateState(batch []ExecutorParams) {
	for _, e := range batch {
		if e.ID > qb.LastID {
			qb.LastID = e.ID
		}
	}
}

func (qb *ExecutorQueryBuilder) ResetState() {}

func (qb *ExecutorQueryBuilder) GetTwoTableSelectQuery() string {
	query := fmt.Sprintf(`
            select id, url
            from %s
            where id > %d
            order by id
            limit %d
    `, qb.SelectionTableName, qb.LastID, qb.BatchSize)
	return query
}

func (qb ExecutorQueryBuilder) GetSelectQuery() string {
	switch qb.Mode {
	case config.TwoTableMode:
		return qb.GetTwoTableSelectQuery()
	default:
		panic("Not implemented")
	}
}
