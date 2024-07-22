package meta

import (
	"fmt"
	"time"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
)

type VerifyQueryBuilder struct {
	Offset        int
	Limit         int
	LastTimestamp time.Time
	Mode          config.RunnerMode
}

func (qb *VerifyQueryBuilder) UpdateState(batch []VerifyParams) {
	for _, el := range batch {
		if el.Timestamp.Sub(qb.LastTimestamp) > 0 {
			qb.LastTimestamp = el.Timestamp
		}
	}
	log.S.Debug(
		"Updating the inner state of the query builder",
		log.L().Tag(log.LogTagApiImpl).
			Add("timestamp", qb.LastTimestamp),
	)
}

func (qb *VerifyQueryBuilder) ResetState() {
	qb.LastTimestamp = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
}

// Implement the [rinterface.StoredValue] interface.
func (qb VerifyQueryBuilder) GetContinuousSelectQuery() string {
	return fmt.Sprintf(`
        with last as (
            select duns, url, max(ts) as max_ts
            from wv.master
            where is_active = True
            group by duns, url
        ),
        batch as (
            select duns, url, max_ts
            from last
            where max_ts < (now() - toIntervalDay(%d)) and max_ts > %d
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
    `, qb.Offset, qb.LastTimestamp.UnixNano(), qb.Limit)
}

func (qb VerifyQueryBuilder) GetSelectQuery() string {
	switch qb.Mode {
	case config.ContinuousMode:
		return qb.GetContinuousSelectQuery()
	default:
		panic("Not implemented")
	}
}
