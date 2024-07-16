package dbclient

import (
	"context"
	"fmt"

	"orb/runner/src/config"
	"orb/runner/src/log"
	ri "orb/runner/src/runner/interface"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type ClickHouseClient[S ri.StoredValue, P ri.StoredParams] struct {
	Connection driver.Conn
}

func NewClickHouseClient[S ri.StoredValue, P ri.StoredParams](
	config config.ClickHouseConfig,
) (
	client *ClickHouseClient[S, P],
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
	return &ClickHouseClient[S, P]{Connection: conn}, version, err
}

func (client *ClickHouseClient[S, P]) AsyncInsertBatch(
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

func (client *ClickHouseClient[S, P]) SelectNextBatch(
	ctx context.Context,
	batchCounter int,
) (result []P, err error) {
	log.S.Debugw("Trying to retrieve a new batch from database")
	var nilInstance P
	var query string
	requestedSize := config.C.Run.RequestBatchSize
	switch config.C.Api.Mode {
	case config.ContiniousMode:
		days := config.C.Run.DayOffset
		rawQuery := nilInstance.GetContiniousSelectQuery()
		query = fmt.Sprintf(rawQuery, days, requestedSize)
	case config.BatchMode:
		offset := requestedSize * batchCounter
		rawQuery := nilInstance.GetSimpleSelectQuery()
		query = fmt.Sprintf(rawQuery, requestedSize, offset)
	default:
		log.S.Panicw("Unexpected mode", "input_value", config.C.Api.Type)
	}
	if err = client.Connection.Select(ctx, &result, query); err != nil {
		return nil, err
	}
	return result, nil
}
