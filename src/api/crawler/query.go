package crawler

import (
	"fmt"
)

type CrawlerQueryBuilder struct {
	Offset int
	Limit  int
}

func (qb CrawlerQueryBuilder) GetTwoTableSelectQuery() string {
	query := fmt.Sprintf(`
        SELECT url from master LIMIT %d OFFSET %d
    `, qb.Limit, qb.Offset)
	qb.Offset += qb.Limit
	return query
}

func (_ CrawlerQueryBuilder) GetContiniousSelectQuery() string {
	panic("Method is not implemented")
}
