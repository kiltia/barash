package crawler

import (
	"fmt"

	"orb/runner/src/config"
)

type CrawlerQueryBuilder struct {
	Offset    int
	BatchSize int
	Mode      config.RunnerMode
}

func (qb *CrawlerQueryBuilder) UpdateState(batch []CrawlerParams) {
	qb.Offset += qb.BatchSize
}

func (qb *CrawlerQueryBuilder) ResetState() {
	qb.Offset = 0
}

func (qb *CrawlerQueryBuilder) GetTwoTableSelectQuery() string {
	query := fmt.Sprintf(`
            select url
            from wv.master
            where is_active = True
            group by url
            limit %d offset %d
    `, qb.BatchSize, qb.Offset)
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
