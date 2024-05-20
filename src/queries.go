package main

const (
	INSERT_BATCH = `
		INSERT INTO master VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,  now())
	`
	SELECT_BATCH = `
            with last as (
                select duns, url, max(ts) as max_ts
                from wv.master
                where is_active = True
                group by duns, url
            ),
            batch as (
                select duns, url, max_ts
                from last
                where max_ts < (NOW() - toIntervalDay(%d))
                limit %d
            ),
            final as (
                select
                    batch.duns as duns,
                    batch.url as url,
                    gdmi.name,
                    gdmi.loc_address1, gdmi.loc_address2, gdmi.loc_city, gdmi.loc_state, gdmi.loc_zip, gdmi.loc_country,
                    gdmi.mail_address1, gdmi.mail_address2, gdmi.mail_city, gdmi.mail_state, gdmi.mail_zip, gdmi.mail_country
                from wv.gdmi_compact gdmi
                inner join batch using (duns)
                where gdmi.duns != '' and batch.url != ''
                order by cityHash64(batch.duns, batch.url)
            )
            select * from final
	`
)
