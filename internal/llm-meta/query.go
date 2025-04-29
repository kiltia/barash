package llmmeta

import (
	"fmt"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
)

type LlmTaskQueryBuilder struct {
	LastId uint64
	Limit  int
	Mode   config.RunnerMode
}

func (qb *LlmTaskQueryBuilder) UpdateState(
	batch []LlmTaskParams,
) {
	for _, p := range batch {
		if p.Id > qb.LastId {
			qb.LastId = p.Id
		}
	}

	log.S.Info(
		"QueryBuilder state was updated",
		log.L().
			Add("last_id", qb.LastId).
			Add("tasks_fetched", len(batch)),
	)
}

func (qb *LlmTaskQueryBuilder) ResetState() {
	qb.LastId = 0
	log.S.Info(
		"QueryBuilder state reset",
		log.L().
			Add("last_id", qb.LastId),
	)
}

func (qb LlmTaskQueryBuilder) GetTwoTableSelectQuery() string {
	query := fmt.Sprintf(
		`
        SELECT
            id, duns, url, name, dba,
            loc_address1, loc_address2,
            loc_city, loc_state,
            loc_zip, loc_country,
            mail_address1, mail_address2,
            mail_city, mail_state,
            mail_zip, mail_country,
            discovery_prompt, task_prompt, json_schema
        FROM %s
        WHERE id > %d
        ORDER BY id
        LIMIT %d
        `,
		config.C.Run.SelectionTableName,
		qb.LastId,
		qb.Limit,
	)
	return query
}

func (qb LlmTaskQueryBuilder) GetSelectQuery() string {
	switch qb.Mode {
	case config.TwoTableMode:
		return qb.GetTwoTableSelectQuery()
	default:
		panic("Not implemented")
	}
}
