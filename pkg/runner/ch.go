package runner

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"go.uber.org/zap"
)

type Clickhouse[S StoredResult, P StoredParams, Q QueryBuilder[P]] struct {
	Conn            driver.Conn
	insertTableName string
}

func NewClickHouseClient[S StoredResult, P StoredParams, Q QueryBuilder[P]](
	host string,
	port string,
	database string,
	username string,
	password string,
	insertTableName string,
) (
	client *Clickhouse[S, P, Q],
	version *proto.ServerHandshake,
	err error,
) {
	var conn driver.Conn
	zap.S().Debug("opening connection to the ClickHouse")
	conn, err = clickhouse.Open(
		&clickhouse.Options{
			Addr: []string{
				fmt.Sprintf(
					"%s:%s",
					host,
					port,
				),
			},
			Auth: clickhouse.Auth{
				Database: database,
				Username: username,
				Password: password,
			},
		},
	)
	if err != nil {
		zap.S().Errorw(
			"opening connection to the ClickHouse",
			"error", err,
		)
		return nil, nil, err
	}
	version, err = conn.ServerVersion()
	if err != nil {
		zap.S().Errorw(
			"retrieving ClickHouse server version",
			"error", err,
		)
		return nil, nil, err
	}
	return &Clickhouse[S, P, Q]{
		Conn:            conn,
		insertTableName: insertTableName,
	}, version, err
}

func (client *Clickhouse[S, P, Q]) InsertBatch(
	ctx context.Context,
	batch []S,
	tag string,
) error {
	zap.S().Debug("inserting a batch to the database")
	query := fmt.Sprintf("INSERT INTO %s", client.insertTableName)
	zap.S().Debugw(
		"sending query to the database",
		"query", query,
	)
	batchBuilder, err := client.Conn.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}
	for i := range batch {
		err := batchBuilder.AppendStruct(&batch[i])
		if err != nil {
			return err
		}
	}
	if err := batchBuilder.Send(); err != nil {
		return err
	}

	return nil
}

func (client *Clickhouse[S, P, Q]) GetNextBatch(
	ctx context.Context,
	sql string,
	queryBuilder Q,
) (result []P, err error) {
	zap.S().Debug("retrieving a new batch from the database")
	query := queryBuilder.FormatQuery(sql)
	zap.S().Debugw(
		"selecting a new batch from the database",
		"query", query,
	)
	err = client.Conn.Select(ctx, &result, query)
	return result, err
}

func (client *Clickhouse[S, P, Q]) InitTable(
	ctx context.Context,
) error {
	var nilInstance S
	return client.Conn.Exec(
		ctx,
		nilInstance.GetCreateQuery(client.insertTableName),
	)
}
