.PHONY: build test run fmt lint tidy

build:
	go build -o timbrex .

test:
	go test ./...

run:
	go run . check examples/hello.tbx

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

# Fail when go.mod or go.sum would change. Does not rewrite them.
tidy:
	go mod tidy -diff
