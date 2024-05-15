package main

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
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
		fmt.Printf(
			"Connection to the ClickHouse database was unsuccessful! Gotten error: %s",
			err,
		)
		return nil, err
	}
	v, err := conn.ServerVersion()
	if err != nil {
		fmt.Printf(
			"Connection to the ClickHouse database was unsuccessful! Gotten error: %s",
			err,
		)
		return nil, err
	}
	fmt.Println("Connection to the ClickHouse database was successful!")
	fmt.Println(v)
	return &ClickHouseClient{Connection: conn}, nil
}

func (client ClickHouseClient) AsyncInsertBatch(
	batch []VerificationResult,
) error {
	ctx := context.Background()
	for i := 0; i < len(batch); i++ {
		verifyParams := batch[i].VerifyParams
		link := batch[i].VerificationLink
		statusCode := batch[i].StatusCode
		response := batch[i].VerificationResponse
		score := response.Score
		componentError := response.Error
		matchMask := response.MatchMask
		matchMaskSummary := matchMask.MatchMaskSummary
		debugInfo := response.DebugInfo
		crawlerDebug := debugInfo.CrawlerDebug
		crawlerErrors := crawlerDebug.CrawlerErrors
		crawlFails := crawlerDebug.CrawlFails
		crawledPages := crawlerDebug.CrawledPages
		failStatus := crawlerDebug.FailStatus
		pageStats := crawlerDebug.PageStats
		numErrors := pageStats.Errors
		numFails := pageStats.Fails
		numSuccesses := pageStats.Successes
		err := client.Connection.AsyncInsert(
			ctx,
			INSERT,
			false,
			verifyParams.Duns,
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
			link,
			componentError,
			failStatus,
			statusCode,
			crawlerErrors,
			crawlFails,
			crawledPages,
			numErrors,
			numFails,
			numSuccesses,
			debugInfo.Features,
			matchMask.MatchMaskDetails,
			matchMaskSummary.Name,
			matchMaskSummary.Address1,
			matchMaskSummary.Address2,
			matchMaskSummary.City,
			matchMaskSummary.State,
			matchMaskSummary.Country,
			matchMaskSummary.DomainNameSimilarity,
			score,
		)
		if err != nil {
			fmt.Printf(
				"Insertion to the ClickHouse database was unsuccessful! Gotten error: %s",
				err,
			)
			return err
		}
	}
	fmt.Println("Insertion to the ClickHouse database was successful!")
	return nil
}
