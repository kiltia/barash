package main

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

type ClickHouseClient struct {
	Connection driver.Conn
}

func NewClickHouseClient(config ClickHouseConfig) (*ClickHouseClient, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
	})
	if err != nil {
		fmt.Printf("Connection to the ClickHouse database was unsuccessful! Gotten error: %s", err)
		return nil, err
	}
	v, err := conn.ServerVersion()
	if err != nil {
		fmt.Printf("Connection to the ClickHouse database was unsuccessful! Gotten error: %s", err)
		return nil, err
	}
	fmt.Println("Connection to the ClickHouse database was successful!")
	fmt.Println(v)
	return &ClickHouseClient{Connection: conn}, nil
}

func (client ClickHouseClient) AsyncInsertBatch(batch []Triple) error {
	ctx := context.Background()
	for i := 0; i < len(batch); i++ {
		id := uuid.New().String()
		verifyParams := batch[i].VerifyParams
		score := batch[i].VerificationResult.Score
		statusCode := batch[i].StatusCode
		err := client.Connection.AsyncInsert(
			ctx,
			INSERT,
			true,
			id,
			verifyParams.Url,
			verifyParams.Name,
			verifyParams.MailAddress1,
			verifyParams.MailAddress2,
			verifyParams.MailCity,
			verifyParams.MailState,
			verifyParams.MailZip,
			verifyParams.MailCountry,
			verifyParams.LocAddress1,
			verifyParams.LocAddress2,
			verifyParams.LocCity,
			verifyParams.LocState,
			verifyParams.LocZip,
			verifyParams.LocCountry,
			score,
			statusCode,
		)
		if err != nil {
			fmt.Printf("Insertion to the ClickHouse database was unsuccessful! Gotten error: %s", err)
			return err
		}
	}
	fmt.Println("Insertion to the ClickHouse database was successful!")
	return nil
}
