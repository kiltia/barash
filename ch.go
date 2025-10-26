package barash

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/kiltia/barash/config"
	"go.uber.org/zap"
)

var (
	_ Sink[StoredResult]   = &ClickhouseSink[StoredResult]{}
	_ Source[StoredParams] = &ClickhouseSource[StoredParams]{}
)

type ClickhouseWrapper struct {
	Conn driver.Conn
}

func NewClickhouseWrapper(
	cfg config.DatabaseConfig,
) (*ClickhouseWrapper, error) {
	conn, err := getConn(cfg)
	if err != nil {
		return nil, err
	}
	return &ClickhouseWrapper{
		Conn: conn,
	}, err
}

func getConn(cfg config.DatabaseConfig) (driver.Conn, error) {
	zap.S().Debug("opening connection to the ClickHouse")
	conn, err := clickhouse.Open(
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
				Username: cfg.Credentials.Username,
				Password: cfg.Credentials.Password,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	version, err := conn.ServerVersion()
	if err != nil {
		zap.S().Errorw(
			"retrieving ClickHouse server version",
			"error", err,
		)
		return nil, err
	}

	zap.S().Debugw(
		"opened connection to the ClickHouse",
		"version", fmt.Sprintf("%v", version),
	)

	return conn, nil
}

type ClickhouseSink[S StoredResult] struct {
	ClickhouseWrapper
	insertTable string
}

type ClickhouseSource[P StoredParams] struct {
	ClickhouseWrapper
	selectTable string
}

func NewClickhouseSink[S StoredResult](
	cfg config.SinkConfig,
) (
	client *ClickhouseSink[S],
	err error,
) {
	w, err := NewClickhouseWrapper(cfg.DatabaseConfig)
	if err != nil {
		return nil, err
	}
	return &ClickhouseSink[S]{
		insertTable:       cfg.InsertTable,
		ClickhouseWrapper: *w,
	}, nil
}

func NewClickhouseSource[P StoredParams](
	cfg config.SourceConfig,
) (
	client *ClickhouseSource[P],
	err error,
) {
	w, err := NewClickhouseWrapper(cfg.DatabaseConfig)
	if err != nil {
		return nil, err
	}
	return &ClickhouseSource[P]{
		selectTable:       cfg.SelectTable,
		ClickhouseWrapper: *w,
	}, nil
}

func (s *ClickhouseSink[S]) InsertBatch(
	ctx context.Context,
	batch []S,
) error {
	zap.S().Debug("inserting a batch to the database")
	query := fmt.Sprintf("INSERT INTO %s", s.insertTable)
	zap.S().Debugw(
		"Sending query to the database",
	)
	batchBuilder, err := s.Conn.PrepareBatch(ctx, query)
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

func (client *ClickhouseSource[P]) GetNextBatch(
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

func (s *ClickhouseSink[S]) InitTable(
	ctx context.Context,
) error {
	var nilInstance S
	return s.Conn.Exec(
		ctx,
		nilInstance.GetCreateQuery(s.insertTable),
	)
}
