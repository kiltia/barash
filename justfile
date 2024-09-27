format:
    @goimports -l -w ./cmd ./pkg ./internal
    @gofumpt -l -w ./cmd ./pkg ./internal
    @golines -w -m 80 ./cmd ./pkg ./internal

check-format:
    @goimports -d ./cmd ./pkg ./internal
    @gofumpt -d ./cmd ./pkg ./internal
    @golines --dry-run -m 80 ./cmd ./pkg ./internal

lint:
    @golangci-lint run ./cmd/... ./pkg/... ./internal/...
    @gocyclo -over 10 .

test:
    @go test ./cmd/... ./pkg/... ./internal/... -v

vet:
    @go vet ./cmd/... ./pkg/... ./internal/...

pre-commit: format lint vet test

hook-setup:
    echo "just pre-commit" > .git/hooks/pre-commit
    chmod u+x .git/hooks/pre-commit

run-meta $CONFIG_FILE="config/meta.prod.yml":
    @go run ./cmd

run-dev $CONFIG_FILE="config/meta.dev.yml":
    @go run ./cmd

run-crawler $CONFIG_FILE="config/crawler.dev.yml":
    @go run ./cmd

run-prod $config_file="configs/prod.yml":
    @go run ./cmd
