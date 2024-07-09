format:
    @goimports -l -w src
    @gofumpt -l -w src
    @golines -w -m 80 src

check-format:
    @goimports -d src
    @gofumpt -d src
    @golines --dry-run -m 80 src

lint:
    @golangci-lint run ./src/...

test:
    @go test ./src/... -v
