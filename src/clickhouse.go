package main

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type StoredValueType interface {
	GenerateInsertQuery() string
	GenerateSelectQuery() string
	// GenerateTableQuery() string
	AsArray() []any
	GetStatusCode() int
	// AdditionalSuccessLogic()
}

type ClickhouseClient[S StoredValueType, P ParamsType] struct {
	Connection driver.Conn
}

func NewClickHouseClient[S StoredValueType, P ParamsType](
	config ClickHouseConfig,
) (*ClickhouseClient[S, P], *proto.ServerHandshake, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	version, err := conn.ServerVersion()
	if err != nil {
		return nil, nil, err
	}
	return &ClickhouseClient[S, P]{Connection: conn}, version, err
}

func (client ClickhouseClient[S, P]) AsyncInsertBatch(
	batch []S,
	tag string,
) error {
	ctx := context.Background()
	// TODO(nrydanov): Find a way to make it not instance-specific
	query := batch[0].GenerateInsertQuery()
	for i := 0; i < len(batch); i++ {
		innerRepr := batch[i].AsArray()
        innerRepr = append(innerRepr, tag)
		err := client.Connection.AsyncInsert(
			// TODO(nrydanov): Add tag somehow
			ctx, query, false, innerRepr...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client ClickhouseClient[S, P]) SelectNextBatch(
	days int,
	selectBatchSize int,
) (*[]P, error) {
	ctx := context.Background()
	var result []P
	var zeroInstance S
	rawQuery := zeroInstance.GenerateSelectQuery()
	query := fmt.Sprintf(rawQuery, days, selectBatchSize)
	if err := client.Connection.Select(ctx, &result, query); err != nil {
		return nil, err
	}
	return &result, nil
}
