package runner

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"github.com/kiltia/runner/pkg/config"
	"go.uber.org/zap"
)

type ClickHouseClient[S StoredResult, P StoredParams, Q QueryBuilder[S, P]] struct {
	Connection         driver.Conn
	insertionTableName string
}

func NewClickHouseClient[S StoredResult, P StoredParams, Q QueryBuilder[S, P]](
	host string,
	port string,
	database string,
	username string,
	password string,
	insertionTableName string,
) (
	client *ClickHouseClient[S, P, Q],
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
	return &ClickHouseClient[S, P, Q]{
		Connection:         conn,
		insertionTableName: insertionTableName,
	}, version, err
}

func (client *ClickHouseClient[S, P, Q]) InsertBatch(
	ctx context.Context,
	batch []S,
	tag string,
) error {
	zap.S().Debug("inserting a batch to the database")
	query := fmt.Sprintf("INSERT INTO %s", client.insertionTableName)
	zap.S().Debugw(
		"sending query to the database",
		"query", query,
	)
	batchBuilder, err := client.Connection.PrepareBatch(ctx, query)
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

func (client *ClickHouseClient[S, P, Q]) SelectNextBatch(
	ctx context.Context,
	queryBuilder Q,
) (result []P, err error) {
	zap.S().Debug("retrieving a new batch from the database")
	query := queryBuilder.GetSelectQuery()
	zap.S().Debugw(
		"selecting a new batch from the database",
		"query", query,
	)
	err = client.Connection.Select(ctx, &result, query)
	return result, err
}

func (r *Runner[S, R, P, Q]) initTable(
	ctx context.Context,
) {
	if r.cfg.Mode == config.ContinuousMode {
		zap.S().
			Infow("running in continuous mode, skipping table initialization")
		return
	}
	var nilInstance S
	err := r.clickHouseClient.Connection.Exec(
		ctx,
		nilInstance.GetCreateQuery(r.cfg.Writer.InsertionTableName),
	)

	if err != nil {
		zap.S().Warnw("table creation script has failed", "error", err)
	} else {
		zap.S().Infow("successfully initialized table for the Runner results")
	}
}
