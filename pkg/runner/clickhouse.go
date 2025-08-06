package runner

import (
	"context"
	"fmt"

	"github.com/kiltia/runner/pkg/config"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"go.uber.org/zap"
)

type ClickHouseClient[S StoredResult, P StoredParams, Q QueryBuilder[S, P]] struct {
	Connection         driver.Conn
	insertionTableName string
	selectRetries      config.SelectRetryConfig
}

func NewClickHouseClient[S StoredResult, P StoredParams, Q QueryBuilder[S, P]](
	host string,
	port string,
	database string,
	username string,
	password string,
	insertionTableName string,
	selectRetries config.SelectRetryConfig,
) (
	client *ClickHouseClient[S, P, Q],
	version *proto.ServerHandshake,
	err error,
) {
	var conn driver.Conn
	zap.S().Debug("Opening connection to the ClickHouse")
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
			"Failed to open a connection to the ClickHouse",
			"error", err,
		)
		return nil, nil, err
	}
	zap.S().Debug("Retrieving server version")
	version, err = conn.ServerVersion()
	if err != nil {
		zap.S().Errorw(
			"Failed to retrieve ClickHouse server version",
			"error", err,
		)
		return nil, nil, err
	}
	return &ClickHouseClient[S, P, Q]{
		Connection:         conn,
		insertionTableName: insertionTableName,
		selectRetries:      selectRetries,
	}, version, err
}

func (client *ClickHouseClient[S, P, Q]) InsertBatch(
	ctx context.Context,
	batch []S,
	tag string,
) error {
	zap.S().Debug("Inserting a batch to the database")
	query := fmt.Sprintf("INSERT INTO %s", client.insertionTableName)
	zap.S().Debugw(
		"Sending query to the database",
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

	zap.S().Debug("Successfully saved batch to the database")
	return nil
}

func (client *ClickHouseClient[S, P, Q]) SelectNextBatch(
	ctx context.Context,
	queryBuilder Q,
) (result []P, err error) {
	zap.S().Debug("Retrieving a new batch from the database")
	query := queryBuilder.GetSelectQuery()
	zap.S().Debugw(
		"Selecting a new batch from the database",
		"query", query,
	)
	for attempt := range client.selectRetries.NumRetries {
		if err = client.Connection.Select(ctx, &result, query); err != nil {
			zap.S().Errorw(
				"Got an error while retrieving records from the database",
				"error", err,
			)
			if attempt < client.selectRetries.NumRetries {
				zap.S().Warnw(
					"Retrying query",
					"retry_number", attempt,
				)
			}
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	zap.S().Debugw(
		"Successfully selected a batch from the database",
		"batch_size", len(result),
	)
	return result, nil
}
