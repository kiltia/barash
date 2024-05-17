package main

const (
	INSERT_BATCH = `
		INSERT INTO go_wv_result2 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())
	`
	SELECT_BATCH = `
		with last as (
			select duns, url, max(ts) as max_ts
			from master
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
			from gdmi_compact
			where duns in (select duns from shuffle)
		)
		select
			duns, url, max_ts, name, dba,
			loc_address1, loc_address2, loc_city, loc_state, loc_zip, loc_country,
			mail_address1, mail_address2, mail_city, mail_state, mail_zip, mail_country
		from shuffle
		inner join gdmi
		on gdmi.duns = shuffle.duns;
	`
)
