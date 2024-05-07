package main

import (
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Runner struct {
	clickHouseConnection driver.Conn
}
