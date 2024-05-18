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
        oldest as (
            select duns, url, max_ts
            from last
            where max_ts > (NOW() - toIntervalDay(%d))
            limit %d
        ),
        gdmi as (
            select *
            from wv.gdmi_compact
            where duns in (select duns from oldest)
        )
        select
            duns, url, gdmi.name,
            gdmi.loc_address1, gdmi.loc_address2, gdmi.loc_city, gdmi.loc_state, gdmi.loc_zip, gdmi.loc_country,
            gdmi.mail_address1, gdmi.mail_address2, gdmi.mail_city, gdmi.mail_state, gdmi.mail_zip, mail_country
        from oldest
        inner join gdmi
        on oldest.duns = gdmi.duns
        order by cityHash64(duns, url);
	`
)
