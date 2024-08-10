package crawler

import (
	"fmt"

	"orb/runner/pkg/config"
)

type CrawlerQueryBuilder struct {
	LastId    int64
	BatchSize int
	Mode      config.RunnerMode
}

func (qb *CrawlerQueryBuilder) UpdateState(batch []CrawlerParams) {
	for _, e := range batch {
		if e.Id > qb.LastId {
			qb.LastId = e.Id
		}
	}
}

func (qb *CrawlerQueryBuilder) ResetState() {}

func (qb *CrawlerQueryBuilder) GetTwoTableSelectQuery() string {
	query := fmt.Sprintf(`
            select id, url
            from crawler_urls
            where id > %d
            order by id
            limit %d
    `, qb.LastId, qb.BatchSize)
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
