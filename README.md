# Barash

This is command line tool for creating workload on different HTTP servers.

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
- Continious mode (for more info, see "Continious mode")

### Continious mode

Sometimes, you need to run your workload for a long time. In our case, we
utilize one Clickhouse table as both input and output data. So, we need to
select `Freshness` parameter, which will be used to filter out records that
are older than `now() - Freshness`. This records will be used as input.


### Timestamp correction

Sometimes, you need to manipulate timestamps that are stored to database.

For example, let's image you have Web Scraper. Some of the requests, of course,
can result in timeouts and result in sudden RPS drop. So, one way to reach
stable RPS is to shuffle timeouts. Adding random duration to each requests
finished with timeout can help to achieve that.

Also, when running in continious mode, you may want to retry requests that
are failed because of client/server errors. This can also be achieved by
correction timestamp like: `current timestamp - freshness + correction value`. For example,
if you consider all results that are older than 7 days, and you want to retry
requests that are failed in 1 day, then you should correct its timestamp like:
`current timestamp - 7 days + 1 day`.

Correction value can be set via `Correction.ErrorCorrection` field in config.


## Usage

Project contains two entry points:
1. `./cmd/main.go`
2. `./cmd/runner-cli/main.go`


First one uses environment variables that are already set to configure the tool.
Second one is CLI configurator that allows to load configurations, edit them
in interactive mode, save them and run with selected configuration.

To run it, you need to have `go` installed and run:

```bash
go run ./cmd/main.go
```

or

```bash
go run ./cmd/runner-cli/main.go
```

You can also install it globally:
```bash
go install ./cmd/runner-cli/main.go
```


## Configuration

You can find all possible configuration options in `pkg/runner/config.go` file or
in CLI configurator mode.

**Important**: application uses following precendency order:
1. Variables that are set in environment or that are present in `.env` file in project root
2. Selected config file

So, if you've set `RUN_MODE` variable in `.env`, it will have higher priority than
variables set in selected configuration file.
