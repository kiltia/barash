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
