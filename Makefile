.PHONY: all build test coverage lint clean

BINARY_NAME=bin/nextcloud-mcp-gateway

all: lint test build

build:
	mkdir -p bin
	go build -v -o $(BINARY_NAME) ./cmd/nextcloud-mcp-gateway

test:
	go test -v -race -cover ./internal/...

coverage:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./internal/...
	go tool cover -func=coverage.txt

lint:
	go vet ./...

clean:
	rm -rf bin coverage.txt *.out
