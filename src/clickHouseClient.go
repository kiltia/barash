package main

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type ClickHouseClient struct {
	Connection driver.Conn
}

func NewClickHouseClient(config ClickHouseConfig) (*ClickHouseClient, *proto.ServerHandshake, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
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
	version, err := conn.ServerVersion()
	if err != nil {
		return nil, nil, err
	}
	return &ClickHouseClient{Connection: conn}, version, err
}

func (client ClickHouseClient) AsyncInsertBatch(
	batch []VerificationResult,
	tag string,
) error {
	ctx := context.Background()
	for i := 0; i < len(batch); i++ {
		verifyParams := batch[i].VerifyParams
		verificationUrl := batch[i].VerificationLink
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
			INSERT_BATCH,
			false,
			verifyParams.Duns,
			true,
			verifyParams.Url,
			verificationUrl,
			statusCode,
			componentError,
			failStatus,
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
			response.FinalUrl,
			score,
			tag,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client ClickHouseClient) SelectNextBatch(days int, selectBatchSize int) (*[]VerifyParams, error) {
	ctx := context.Background()
	var result []VerifyParams
	query := fmt.Sprintf(SELECT_BATCH, days, selectBatchSize)
	if err := client.Connection.Select(ctx, &result, query); err != nil {
		return nil, err
	}
	return &result, nil
}
