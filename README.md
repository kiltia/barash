# Barash

This is a library for creating workload on different HTTP servers. Just implement interfaces, create `Runner` instance and you're ready to go.

**Note**: This project is in early stage of development and probably not ready
for production use.

## Features

Basic features:
- Creating concurrent requests to HTTP servers
- Saving results to database (currently, only Clickhouse is supported for historical reasons)

Advanced features:
- Increasing number of concurrent requests (also known as "Warm up")
- Circuit breaker pattern, which allows to temporarily stop requests
- No data loss on shutdown (except for requests that are in progress)
- Timestamp correction (for more info, see "Timestamp correction")
- Retries with backoff, timeouts
- Continuous mode (for more info, see "Continuous mode")

### Continuous mode

Sometimes, you need to run your workload for a long time. In our case, we
utilize one Clickhouse table as both input and output data. So, we need to
select `Freshness` parameter, which will be used to filter out records that
are older than `now() - Freshness`. These records will be used as input.

### Timestamp correction

Sometimes, you need to manipulate timestamps that are stored to database.

For example, let's imagine you have Web Scraper. Some of the requests, of course,
can result in timeouts and result in sudden RPS drop. So, one way to reach
stable RPS is to shuffle timeouts. Adding random duration to each request
finished with timeout can help to achieve that.

Also, when running in continuous mode, you may want to retry requests that
are failed because of client/server errors. This can also be achieved by
correction timestamp like: `current timestamp - freshness + correction value`. For example,
if you consider all results that are older than 7 days, and you want to retry
requests that are failed in 1 day, then you should correct its timestamp like:
`current timestamp - 7 days + 1 day`.

Correction value can be set via `correction.error_correction` field in config.

## Usage

### Running the service

To run the service, you have several options for configuration:

1. **Using YAML configuration file:**
```bash
go run ./cmd/main.go -config config.yaml
```

2. **Using environment variables only:**
```bash
go run ./cmd/main.go -env
```

3. **Default behavior (environment variables):**
```bash
go run ./cmd/main.go
```

You can also build and install the binary:
```bash
go build -o barash ./cmd/main.go
./barash -config config.yaml
```

## Configuration

The configuration system supports both YAML files and environment variables. Environment variables take precedence over YAML configuration, allowing you to override specific settings without modifying the configuration file.

### Configuration Structure

The configuration is organized into the following sections:

#### API Configuration (`api`)
Settings for the external API endpoints:
```yaml
api:
  type: "rest"
  host: "api.example.com"
  port: "443"
  scheme: "https"
  endpoint: "/v1/data"
  method: "POST"
  api_timeout: "3m"
  num_retries: 3
  min_wait_time: "2s"
  max_wait_time: "16s"
  extra_params:
    format: "json"
  body_file_path: "request_body.json"
```

#### Provider Configuration (`provider`)
Settings for data retrieval from the database:
```yaml
provider:
  sleep_time: "1m"
  select_batch_size: 40000
  select_table: "source_table"
  select_retries: 5
  select_sql_path: "select.sql"
  
  source:
    backend: "ch"  # "ch" or "pg"
    credentials:
      database: "source_db"
      username: "user"
      password: "password"
      host: "127.0.0.1"
      port: "9000"
  
  continuous_mode:
    freshness: "168h"  # 7 days
```

#### Fetcher Configuration (`fetcher`)
Settings for API request execution with circuit breaker:
```yaml
fetcher:
  min_fetcher_workers: 400
  max_fetcher_workers: 800
  duration: "60s"
  enable_warmup: false
  idle_time: "10s"
  timeout: "40s"
  
  circuit_breaker:
    enabled: true
    max_requests: 10
    consecutive_failure: 10
    total_failure_per_interval: 900
    interval: "60s"
    timeout: "360s"
```

#### Writer Configuration (`writer`)
Settings for saving results to the database with correction logic:
```yaml
writer:
  insert_batch_size: 10000
  insert_table: "results_table"
  insert_sql_path: "insert.sql"
  save_tag: "production"
  
  sink:
    backend: "ch"  # "ch" or "postgres"
    credentials:
      database: "sink_db"
      username: "user"
      password: "password"
      host: "127.0.0.1"
      port: "9000"
  
  correction:
    enable_errors_correction: false
    error_correction: "24h"
    enable_timeouts_correction: true
    max_timeout_correction: "504h"
```

#### Log Configuration (`log`)
Logging settings:
```yaml
log:
  level: "info"      # debug, info, warn, error
  encoding: "json"   # json or console
```

#### Shutdown Configuration (`shutdown`)
Graceful shutdown settings:
```yaml
shutdown:
  grace_period: "60s"
  db_save_timeout: "30s"
```

### Configuration Files

You can find a complete example configuration in `config.example.yaml`. Copy this file and modify it according to your needs:

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your settings
```

### Environment Variables

All configuration options can be overridden using environment variables. The variable names follow this pattern:

- `RUN_MODE` for the run mode
- `API_HOST`, `API_PORT`, etc. for API configuration
- `PROVIDER_SLEEP_TIME`, `PROVIDER_SELECT_BATCH_SIZE`, etc. for provider configuration
- `FETCHER_MIN_WORKERS`, `FETCHER_MAX_WORKERS`, etc. for fetcher configuration
- `WRITER_INSERT_BATCH_SIZE`, `WRITER_INSERT_TABLE`, etc. for writer configuration
- `CB_ENABLED`, `CB_MAX_REQUESTS`, etc. for circuit breaker configuration
- `CONTINUOUS_FRESHNESS` for continuous mode configuration
- `CORRECTION_ENABLE_ERRORS`, etc. for correction configuration
- `LOG_LEVEL`, `LOG_ENCODING` for logging configuration
- `SHUTDOWN_GRACE_PERIOD`, `SHUTDOWN_DB_SAVE_TIMEOUT` for shutdown configuration

### Loading Order

The configuration loading follows this precedence (highest to lowest):

1. Environment variables
2. YAML configuration file
3. Default values

This means environment variables will always override YAML settings, allowing for flexible deployment configurations.

## Development

### Prerequisites

- Go 1.21 or later
- Access to ClickHouse or PostgreSQL database

### Building

```bash
go build ./cmd/main.go
```

### Testing

```bash
go test ./...
```

### Code formatting

```bash
go mod tidy
go fmt ./...
```
