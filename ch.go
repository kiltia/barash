package barash

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"github.com/kiltia/barash/config"
	"go.uber.org/zap"
)

type Clickhouse[S StoredResult, P StoredParams] struct {
	Conn            driver.Conn
	insertTableName string
}

func NewClickHouseClient[S StoredResult, P StoredParams](
	cfg config.DatabaseConfig,
) (
	client *Clickhouse[S, P],
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
					cfg.Host,
					cfg.Port,
				),
			},
			Auth: clickhouse.Auth{
				Database: cfg.Database,
				Username: cfg.Username,
				Password: cfg.Password,
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
	return &Clickhouse[S, P]{
		Conn: conn,
	}, version, err
}

func (client *Clickhouse[S, P]) InsertBatch(
	ctx context.Context,
	batch []S,
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

	return batchBuilder.Send()
}

func (client *Clickhouse[S, P]) GetNextBatch(
	ctx context.Context,
	sql string,
	queryBuilder QueryState[P],
) (result []P, err error) {
	zap.S().Debug("retrieving a new batch from the database")
	tmpl, err := template.New("query").Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("parsing sql: %w", err)
	}

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, queryBuilder); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	query := buf.String()

	zap.S().Debugw(
		"selecting a new batch from the database",
		"query", query,
	)
	return result, client.Conn.Select(ctx, &result, query)
}

func (client *Clickhouse[S, P]) InitTable(
	ctx context.Context,
) error {
	var nilInstance S
	return client.Conn.Exec(
		ctx,
		nilInstance.GetCreateQuery(client.insertTableName),
	)
}
