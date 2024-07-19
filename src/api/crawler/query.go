package crawler

import (
	"fmt"
	"orb/runner/src/config"
)

type CrawlerQueryBuilder struct {
	Offset    int
	BatchSize int
    Mode config.RunnerMode
}

func (qb *CrawlerQueryBuilder) UpdateState(batch []CrawlerParams) {}

func (qb *CrawlerQueryBuilder) ResetState() {
	qb.Offset = 0
}

func (qb *CrawlerQueryBuilder) GetTwoTableSelectQuery() string {
	query := fmt.Sprintf(`
        SELECT url from master LIMIT %d OFFSET %d
    `, qb.BatchSize, qb.Offset)
	qb.Offset += qb.BatchSize
	return query
}

func (qb CrawlerQueryBuilder) GetSelectQuery() string {
	switch qb.Mode {
    case config.TwoTableMode:
        return qb.GetTwoTableSelectQuery()
    default:
        panic("Not implemented")
    }
}
