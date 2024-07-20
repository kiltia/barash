package dbclient

import (
	"context"
	"fmt"

	"orb/runner/pkg/config"
	"orb/runner/pkg/log"
	ri "orb/runner/pkg/runner/interface"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type ClickHouseClient[S ri.StoredValue, P ri.StoredParams, Q ri.QueryBuilder[S, P]] struct {
	Connection driver.Conn
}

func NewClickHouseClient[S ri.StoredValue, P ri.StoredParams, Q ri.QueryBuilder[S, P]](
	config config.ClickHouseConfig,
) (
	client *ClickHouseClient[S, P, Q],
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
	return &ClickHouseClient[S, P, Q]{Connection: conn}, version, err
}

func (client *ClickHouseClient[S, P, Q]) AsyncInsertBatch(
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

func (client *ClickHouseClient[S, P, Q]) SelectNextBatch(
	ctx context.Context,
	queryBuilder Q,
) (result []P, err error) {
	log.S.Debugw("Trying to retrieve a new batch from database")
	query := queryBuilder.GetSelectQuery()
	log.S.Debugw("Sending query to database", "query", query)
	if err = client.Connection.Select(ctx, &result, query); err != nil {
		log.S.Error(
			"Got an error while retrieving records from database",
			"error",
			err,
		)
		return nil, err
	}
	log.S.Debugw("Successfully got records from database", "count", len(result))
	return result, nil
}
