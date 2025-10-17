format:
    go tool goimports -l -w .
    go tool gofumpt -l -w .
    go tool golines -w -m 80 .

lint:
    golangci-lint run ./...
    go vet ./...

test:
    go test ./...

unit-test:
    go test ./internal/... ./pkg/... ./cmd/...

pre-commit: format lint test

hook-setup:
    echo "just pre-commit" > .git/hooks/pre-commit
    chmod u+x .git/hooks/pre-commit
