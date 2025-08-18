format:
    goimports -l -w ./...
    gofumpt -l -w ./...
    golines -w -m 80 ./...

lint:
    golangci-lint run ./...
    gocyclo -over 10 .

test:
    go test ./...

unit-test:
    go test ./internal/... ./pkg/... ./cmd/...

pre-commit: format lint vet test

hook-setup:
    echo "just pre-commit" > .git/hooks/pre-commit
    chmod u+x .git/hooks/pre-commit
