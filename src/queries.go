package main

const (
	INSERT_BATCH = `
		INSERT INTO master VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,  now())
	`
	SELECT_BATCH = `
		with last as (
		    select duns, url, max(ts) as max_ts
		    from wv.master
		    where is_duns_active
		    group by duns, url
		),
		oldest as (
		    select duns, url, max_ts
		    from last
		    where max_ts > (NOW() - toIntervalDay(%d))
		    order by max_ts
		    limit %d
		),
		shuffle as (
		    select * from oldest
		    order by cityHash64(duns, url)
		),
		gdmi as (
		    select *
		    from wv.gdmi_compact
		    where duns in (select duns from shuffle)
		)
		select
		    duns, url, name,
		    loc_address1, loc_address2, loc_city, loc_state, loc_zip, loc_country,
		    mail_address1, mail_address2, mail_city, mail_state, mail_zip, mail_country
		from shuffle
		left join gdmi
		on gdmi.duns = shuffle.duns;
	`
)
