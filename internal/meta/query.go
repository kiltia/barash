package meta

import (
	"fmt"
	"time"

	"github.com/kiltia/runner/pkg/config"

	"go.uber.org/zap"
)

const DateFormat = "2006-01-02 15:04:05.999999"

type VerifyQueryBuilder struct {
	Interval           time.Duration
	Limit              int
	StartTimestamp     time.Time
	LastTimestamp      time.Time
	LastURL            string
	LastDuns           string
	Mode               config.RunnerMode
	SelectionTableName string
}

func (qb *VerifyQueryBuilder) UpdateState(
	batch []VerifyParams,
) {
	for _, p := range batch {
		if p.Timestamp.After(
			qb.LastTimestamp,
		) {
			qb.LastTimestamp = p.Timestamp
		}
		qb.LastURL = p.URL
		qb.LastDuns = p.Duns
	}

	zap.S().Infow(
		"QueryBuilder state was updated",
		"last_ts", qb.LastTimestamp.String(),
		"start_ts", qb.StartTimestamp.String(),
		"tasks_fetched", len(batch),
	)
}

func (qb *VerifyQueryBuilder) ResetState() {
	qb.StartTimestamp = time.Now().UTC()
	qb.LastTimestamp = time.Unix(0, 1).UTC()
	zap.S().Infow(
		"QueryBuilder state reset",
		"last_ts", qb.LastTimestamp.String(),
		"start_ts", qb.StartTimestamp.String(),
	)
}

// Implement the [rinterface.StoredValue] interface.
func (qb VerifyQueryBuilder) GetContinuousSelectQuery() string {
	return fmt.Sprintf(
		`
        with last as (
            select duns, url, maxMerge(max_ts) as max_ts
            from %s
            group by duns, url
        ),
        ordered as (
            select duns, url, max_ts
            from last
            where max_ts < toDateTime64('%s', 6) - toIntervalSecond(%d) and max_ts > toDateTime64('%s', 6)
        ),
        final as (
            select
                ordered.duns as duns,
                ordered.url as url,
                ordered.max_ts as ts,
                gdmi.name,
                gdmi.loc_address1, gdmi.loc_address2,
                gdmi.loc_city, gdmi.loc_state,
                gdmi.loc_zip, gdmi.loc_country,
                gdmi.mail_address1, gdmi.mail_address2,
                gdmi.mail_city, gdmi.mail_state,
                gdmi.mail_zip, gdmi.mail_country,
                gdmi.dba
            from wv.gdmi_compact gdmi
            inner join ordered using (duns)
            order by ordered.max_ts
			limit %d
        )
        select * from final
    `,
		qb.SelectionTableName,
		qb.StartTimestamp.Format(DateFormat),
		int(qb.Interval.Seconds()),
		qb.LastTimestamp.Format(DateFormat),
		qb.Limit,
	)
}

func (qb VerifyQueryBuilder) GetTwoTableSelectQuery() string {
	query := fmt.Sprintf(
		`
        SELECT
            duns, url, name, dba,
            loc_address1, loc_address2,
            loc_city, loc_state,
            loc_zip, loc_country,
            mail_address1, mail_address2,
            mail_city, mail_state,
            mail_zip, mail_country FROM %s
        WHERE ('%s', '%s') < (duns, url)
        ORDER BY (duns, url)
        LIMIT %d
        `,
		qb.SelectionTableName,
		qb.LastDuns,
		qb.LastURL,
		qb.Limit,
	)
	return query
}

func (qb VerifyQueryBuilder) GetSelectQuery() string {
	switch qb.Mode {
	case config.ContinuousMode:
		return qb.GetContinuousSelectQuery()
	case config.TwoTableMode:
		return qb.GetTwoTableSelectQuery()
	default:
		panic("Not implemented")
	}
}
