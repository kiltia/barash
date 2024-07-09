package main

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type ClickhouseClient[S StoredValueType, P ParamsType] struct {
	Connection driver.Conn
}

func NewClickHouseClient[S StoredValueType, P ParamsType](
	config ClickHouseConfig,
) (
	client *ClickhouseClient[S, P],
	version *proto.ServerHandshake,
	err error,
) {
	var conn driver.Conn
	conn, err = clickhouse.Open(&clickhouse.Options{
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
	version, err = conn.ServerVersion()
	if err != nil {
		return nil, nil, err
	}
	return &ClickhouseClient[S, P]{Connection: conn}, version, err
}

func (client *ClickhouseClient[S, P]) AsyncInsertBatch(
	ctx context.Context,
	batch []S,
	tag string,
) error {
	var zeroInstance S
	query := zeroInstance.GetInsertQuery()
	for i := 0; i < len(batch); i++ {
		innerRepr := batch[i].AsArray()
		innerRepr = append(innerRepr, tag)
		err := client.Connection.AsyncInsert(
			ctx, query, false, innerRepr...,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *ClickhouseClient[S, P]) SelectNextBatch(
	ctx context.Context,
	days int,
	selectBatchSize int,
) (result []P, err error) {
	var nilInstance S
	rawQuery := nilInstance.GetSelectQuery()
	query := fmt.Sprintf(rawQuery, days, selectBatchSize)
	if err = client.Connection.Select(ctx, &result, query); err != nil {
		return nil, err
	}
	return result, nil
}
