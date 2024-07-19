package crawler

import (
	"fmt"
)

type CrawlerQueryBuilder struct {
	Offset    int
	BatchSize int
}

func (qb *CrawlerQueryBuilder) UpdateState(batch []CrawlerRequest) {}

func (qb *CrawlerQueryBuilder) ResetState() {
	qb.Offset = 0
}

func (qb CrawlerQueryBuilder) GetTwoTableSelectQuery() string {
	query := fmt.Sprintf(`
        SELECT url from master LIMIT %d OFFSET %d
    `, qb.BatchSize, qb.Offset)
	qb.Offset += qb.BatchSize
	return query
}

func (_ CrawlerQueryBuilder) GetContiniousSelectQuery() string {
	panic("Method is not implemented")
}
