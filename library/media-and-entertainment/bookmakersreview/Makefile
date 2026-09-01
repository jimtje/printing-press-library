.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/bookmakersreview-pp-cli$(BIN_EXT) ./cmd/bookmakersreview-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/bookmakersreview-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/bookmakersreview-pp-mcp$(BIN_EXT) ./cmd/bookmakersreview-pp-mcp

install-mcp:
	go install ./cmd/bookmakersreview-pp-mcp

build-all: build build-mcp
