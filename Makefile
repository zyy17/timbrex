.PHONY: build test run fmt lint tidy editor-deps test-editor verify

build:
	go build -o timbrex .

test: build
	go test ./...

run:
	go run . check examples/hello.tbx

fmt:
	gofmt -w .

# Same checks as CI. The pinned linter is golangci-lint v2.14.0.
lint:
	golangci-lint run ./...
	golangci-lint fmt --diff

# Fail when go.mod or go.sum would change. Does not rewrite them.
tidy:
	go mod tidy -diff

# Install editor test dependencies separately from offline verification.
editor-deps:
	npm --prefix editors/vscode ci --ignore-scripts

test-editor:
	npm --prefix editors/vscode test

# All CI checks; install Go modules and editor dependencies first.
verify: tidy test lint test-editor
