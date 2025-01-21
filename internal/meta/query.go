package meta

import (
	"fmt"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
)

const DateFormat = "2006-01-02 15:04:05.999999"

type VerifyQueryBuilder struct {
	DayInterval    int
	Limit          int
	StartTimestamp time.Time
	LastTimestamp  time.Time
	LastUrl        string
	LastDuns       string
	Mode           config.RunnerMode
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
		qb.LastUrl = p.Url
		qb.LastDuns = p.Duns
	}

	log.S.Debug(
		"QueryBuilder state was updated",
		log.L().
			Add("last_ts", qb.LastTimestamp.String()).
			Add("start_ts", qb.StartTimestamp.String()),
	)
}

func (qb *VerifyQueryBuilder) ResetState() {
	qb.StartTimestamp = time.Now().UTC()
}

// Implement the [rinterface.StoredValue] interface.
func (qb VerifyQueryBuilder) GetContinuousSelectQuery() string {
	return fmt.Sprintf(
		`
        with last as (
            select duns, url, maxMerge(max_ts) as max_ts
            from wv.master_aggregated ma
            group by duns, url
        ),
        batch as (
            select duns, url, max_ts
            from last
            where max_ts < toDateTime64('%s', 6) - toIntervalDay(%d) and max_ts >= toDateTime64('%s', 6)
            order by max_ts asc
            limit %d
        ),
        final as (
            select
                batch.duns as duns,
                batch.url as url,
                batch.max_ts as ts,
                gdmi.name,
                gdmi.loc_address1, gdmi.loc_address2,
                gdmi.loc_city, gdmi.loc_state,
                gdmi.loc_zip, gdmi.loc_country,
                gdmi.mail_address1, gdmi.mail_address2,
                gdmi.mail_city, gdmi.mail_state,
                gdmi.mail_zip, gdmi.mail_country,
                gdmi.dba
            from wv.gdmi_compact gdmi
            inner join batch using (duns)
            where gdmi.duns != '' and batch.url != ''
            order by cityHash64(batch.duns, batch.url)
        )
        select * from final
    `,
		config.C.Run.SelectionTableName,
		qb.StartTimestamp.Format(DateFormat),
		qb.DayInterval,
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
		config.C.Run.SelectionTableName,
		qb.LastDuns,
		qb.LastUrl,
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
