package meta

import (
	"fmt"
	"time"

	"orb/runner/pkg/config"
)

type VerifyQueryBuilder struct {
	DayInterval    int
	Limit          int
	StartTimestamp time.Time
	LastTimestamp  time.Time
	LastUrl        string
	LastDuns       string
	Mode           config.RunnerMode
}

func (qb *VerifyQueryBuilder) UpdateState(batch []VerifyParams) {
	for _, p := range batch {
		if p.Timestamp.After(qb.LastTimestamp) {
			qb.LastTimestamp = p.Timestamp
		}
		qb.LastUrl = p.Url
		qb.LastDuns = p.Duns
	}
}

func (qb *VerifyQueryBuilder) ResetState() {
	qb.StartTimestamp = time.Now()
	qb.LastTimestamp = time.Unix(0, 1)
}

// Implement the [rinterface.StoredValue] interface.
func (qb VerifyQueryBuilder) GetContinuousSelectQuery() string {
	return fmt.Sprintf(`
        with last as (
            select duns, url, max(ts64) as max_ts
            from %s
            where is_active = True
            group by duns, url
        ),
        batch as (
            select duns, url, max_ts
            from last
            where max_ts < fromUnixTimestamp64Micro(%d) - toIntervalDay(%d) and max_ts >= fromUnixTimestamp64Micro(%d)
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

            from wv.gdmi_compact gdmi
            inner join batch using (duns)
            where gdmi.duns != '' and batch.url != ''
            order by cityHash64(batch.duns, batch.url)
        )
        select * from final
    `, config.C.Run.SelectionTableName, qb.StartTimestamp.UnixMicro(), qb.DayInterval, qb.LastTimestamp.UnixMicro(), qb.Limit)
}

func (qb VerifyQueryBuilder) GetTwoTableSelectQuery() string {
	query := fmt.Sprintf("SELECT * FROM %s ", config.C.Run.SelectionTableName)
	if qb.LastDuns != "" && qb.LastUrl != "" {
		query += fmt.Sprintf(
			"WHERE cityHash64(%s, %s) < cityHash64(duns, url) ",
			qb.LastDuns,
			qb.LastUrl,
		)
	}
	query += fmt.Sprintf("LIMIT %d", qb.Limit)
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
