package barash

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"github.com/kiltia/barash/config"

	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"
	"resty.dev/v3"
)

type ContextKey int

const (
	ContextKeyFetcherNum ContextKey = iota
)

type Runner[S StoredResult, R Response[S, P], P StoredParams, Q QueryState[P]] struct {
	sinks          []Sink[S]
	src            Source[P]
	httpClient     *resty.Client
	cfg            *config.Config
	circuitBreaker *gobreaker.CircuitBreaker[*resty.Response]
	queryBuilder   Q

	selectSQL string
}

func New[
	S StoredResult,
	R Response[S, P],
	P StoredParams,
	Q QueryState[P],
](
	cfg *config.Config,
	qb Q,
) (*Runner[S, R, P, Q], error) {
	sinks, err := initSinks[S](cfg.Writer.Sinks)
	if err != nil {
		return nil, fmt.Errorf("initializing sinks: %w", err)
	}
	source, err := initSource[P](cfg.Provider.Source)
	if err != nil {
		return nil, err
	}

	httpClient := resty.New().
		SetRetryCount(cfg.API.NumRetries).
		SetTimeout(cfg.API.APITimeout).
		SetRetryWaitTime(cfg.API.MinWaitTime).
		SetRetryMaxWaitTime(cfg.API.MaxWaitTime).
		AddRetryConditions(func(r *resty.Response, err error) bool {
			ctx := r.Request.Context()
			fetcherNum := ctx.Value(ContextKeyFetcherNum).(int)
			if r.StatusCode() >= 500 {
				zap.S().
					Debugw(
						"retrying request",
						"fetcher_num",
						fetcherNum,
						"status_code",
						r.StatusCode(),
						"url",
						r.Request.URL,
					)
				return true
			}
			return false
		}).SetLogger(zap.S())

	selectSQL, err := os.ReadFile(cfg.Provider.Source.SelectSQLPath)
	if err != nil {
		return nil, fmt.Errorf("reading select sql statement: %w", err)
	}

	runner := Runner[S, R, P, Q]{
		httpClient: httpClient,
		src:        source,
		cfg:        cfg,
		// TODO(nrydanov): Remove hardcode when others backends become available
		sinks:        sinks,
		selectSQL:    string(selectSQL),
		queryBuilder: qb,
	}

	runner.circuitBreaker = gobreaker.NewCircuitBreaker[*resty.Response](
		gobreaker.Settings{
			Name:        "outgoing_requests",
			MaxRequests: cfg.Fetcher.CircuitBreaker.MaxRequests,
			Interval:    cfg.Fetcher.CircuitBreaker.Interval,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				if cfg.Fetcher.CircuitBreaker.Enabled {
					tooManyTotal := counts.TotalFailures > cfg.Fetcher.CircuitBreaker.TotalFailurePerInterval
					tooManyConsecutive := counts.ConsecutiveFailures > cfg.Fetcher.CircuitBreaker.ConsecutiveFailure
					return tooManyTotal || tooManyConsecutive
				} else {
					return false
				}
			},
		})
	return &runner, nil
}

// Run the runner's job within a given context.
func (r *Runner[S, R, P, Q]) Run(
	ctx context.Context,
	globalWg *sync.WaitGroup,
) {
	// initialize storage in two-table mode
	err := r.initTable(ctx)
	if err != nil {
		zap.S().
			Warnw("one or more table creation scripts have failed", "error", err)
	} else {
		zap.S().Infow("successfully initialized table for the Runner results")
	}

	tasks := r.startProvider(globalWg, ctx)
	results := r.startFetchers(globalWg, ctx, tasks)
	r.startWriter(globalWg, results)
}

func (r *Runner[S, R, P, Q]) initTable(
	ctx context.Context,
) error {
	if r.cfg.Mode == config.ContinuousMode {
		zap.S().
			Infow("running in continuous mode, skipping table initialization")
		return nil
	}
	var errs []error
	for _, sink := range r.sinks {
		err := sink.InitTable(
			ctx,
		)
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func loadCreds(backend string) (*config.DatabaseCredentials, error) {
	userEnv := fmt.Sprintf("%s_USER", strings.ToUpper(backend))
	passwordEnv := fmt.Sprintf("%s_PASSWORD", strings.ToUpper(backend))
	zap.S().
		Infow("loading credentials", "userEnv", userEnv, "passwordEnv", passwordEnv)
	user := os.Getenv(fmt.Sprintf("%s_USER", strings.ToUpper(backend)))
	password := os.Getenv(fmt.Sprintf("%s_PASSWORD", strings.ToUpper(backend)))
	decodedUser, err := base64.StdEncoding.DecodeString(user)
	if err != nil {
		return nil, fmt.Errorf("decoding username: %w", err)
	}
	decodedPassword, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		return nil, fmt.Errorf("decoding password: %w", err)
	}
	creds := &config.DatabaseCredentials{
		Username: string(decodedUser),
		Password: string(decodedPassword),
	}
	zap.S().Infow("loaded credentials")
	return creds, nil
}

func initSinks[S StoredResult](cfgs []config.SinkConfig) ([]Sink[S], error) {
	var clients []Sink[S]
	var errs []error
	for _, cfg := range cfgs {
		creds, err := loadCreds(cfg.Backend)
		if err != nil {
			errs = append(
				errs,
				fmt.Errorf(
					"loading credentials for backend %s: %w",
					cfg.Backend,
					err,
				),
			)
			continue
		}
		var client Sink[S]
		cfg.Credentials = *creds
		switch cfg.Backend {
		case config.BackendClickhouse:
			var err error
			var version *proto.ServerHandshake
			client, version, err = NewClickhouseSink[S](
				cfg,
			)
			if err != nil {
				errs = append(
					errs,
					fmt.Errorf("initializing %s sink: %w", cfg.Backend, err),
				)
				continue
			}
			zap.S().Infow(
				"created a new clickhouse client",
				"version", fmt.Sprintf("%v", version.Version),
			)
		default:
			zap.S().Fatalw("unknown source backend", "backend", cfg)
		}
		clients = append(clients, client)
	}
	return clients, errors.Join(errs...)
}

func initSource[P StoredParams](cfg config.SourceConfig) (Source[P], error) {
	creds, err := loadCreds(cfg.Backend)
	if err != nil {
		return nil, fmt.Errorf("initializing %s source: %w", cfg.Backend, err)
	}
	var client Source[P]
	cfg.Credentials = *creds
	switch cfg.Backend {
	case config.BackendClickhouse:
		var err error
		var version *proto.ServerHandshake
		client, version, err = NewClickhouseSource[P](
			cfg,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"initializing %s source: %w",
				cfg.Backend,
				err,
			)
		}
		zap.S().Infow(
			"created a new clickhouse client",
			"version", fmt.Sprintf("%v", version.Version),
		)
	default:
		zap.S().Fatalw("unknown source backend", "backend", cfg)
	}
	return client, nil
}
