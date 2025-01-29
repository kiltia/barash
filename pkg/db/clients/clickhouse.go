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
	host string,
	port string,
	database string,
	username string,
	password string,
) (
	client *ClickHouseClient[S, P, Q],
	version *proto.ServerHandshake,
	err error,
) {
	var conn driver.Conn
	log.S.Debug(
		"Opening connection to the ClickHouse",
		log.L().
			Tag(log.LogTagClickHouse),
	)
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
		log.S.Error(
			"Failed to open a connection to the ClickHouse",
			log.L().
				Tag(log.LogTagClickHouse).
				Error(err),
		)
		return nil, nil, err
	}
	log.S.Debug(
		"Retrieving server version",
		log.L().
			Tag(log.LogTagClickHouse),
	)
	version, err = conn.ServerVersion()
	if err != nil {
		log.S.Error(
			"Failed to retrieve ClickHouse server version",
			log.L().
				Tag(log.LogTagClickHouse).
				Error(err),
		)
		return nil, nil, err
	}
	return &ClickHouseClient[S, P, Q]{
		Connection: conn,
	}, version, err
}

func (client *ClickHouseClient[S, P, Q]) InsertBatch(
	ctx context.Context,
	batch []S,
	tag string,
) error {
	log.S.Debug(
		"Inserting a batch to the database",
		log.L().
			Tag(log.LogTagClickHouse),
	)
	query := fmt.Sprintf("INSERT INTO %s", config.C.Run.InsertionTableName)
	log.S.Debug(
		"Sending query to the database",
		log.L().
			Tag(log.LogTagClickHouse).
			Add("query", query),
	)
	batchBuilder, err := client.Connection.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}
	for i := 0; i < len(batch); i++ {
		err := batchBuilder.AppendStruct(&batch[i])
		if err != nil {
			return err
		}
	}
	if err := batchBuilder.Send(); err != nil {
		return err
	}

	log.S.Debug(
		"Successfully saved batch to the database",
		log.L().
			Tag(log.LogTagClickHouse),
	)
	return nil
}

func (client *ClickHouseClient[S, P, Q]) SelectNextBatch(
	ctx context.Context,
	queryBuilder Q,
) (result []P, err error) {
	log.S.Debug(
		"Retrieving a new batch from the database",
		log.L().
			Tag(log.LogTagClickHouse),
	)
	query := queryBuilder.GetSelectQuery()
	log.S.Debug(
		"Sending query to the database",
		log.L().
			Tag(log.LogTagClickHouse).
			Add("query", query),
	)
	for attempt := 0; attempt < config.C.SelectRetries.NumRetries; attempt++ {
		if err = client.Connection.Select(ctx, &result, query); err != nil {
			log.S.Error(
				"Got an error while retrieving records from the database",
				log.L().
					Tag(log.LogTagClickHouse).
					Error(err),
			)
			if attempt < config.C.SelectRetries.NumRetries {
				log.S.Warn(
					"Retrying query",
					log.L().
						Tag(log.LogTagClickHouse).
						Add("attempt", attempt+1),
				)
				continue
			}
			return nil, err
		} else {
			break
		}
	}
	if err = client.Connection.Select(ctx, &result, query); err != nil {
		log.S.Error(
			"Got an error while retrieving records from the database",
			log.L().
				Tag(log.LogTagClickHouse).
				Error(err),
		)
		return nil, err
	}
	log.S.Debug(
		"Successfully got records from the database",
		log.L().
			Tag(log.LogTagClickHouse).
			Add("count", len(result)),
	)
	return result, nil
}
