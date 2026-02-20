.PHONY: build test lint generate clean install

BINARY   := stravacli
VERSION  := $(shell git describe --tags --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-X main.version=$(VERSION)"

## build: compile the binary to ./strava
build:
	go build $(LDFLAGS) -o $(BINARY) .

## install: install the binary to $GOPATH/bin
install:
	go install $(LDFLAGS) .

## test: run all tests with race detector
test:
	go test -race ./...

## lint: run golangci-lint (install if missing)
lint:
	@which golangci-lint >/dev/null 2>&1 || \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
	golangci-lint run ./...

## generate: regenerate the OpenAPI client from strava.minimal.json
generate:
	oapi-codegen -config oapi-codegen.yaml strava.minimal.json

## snapshot: build a local release snapshot (no git tag required)
snapshot:
	goreleaser release --snapshot --clean

## release: publish a tagged release to GitHub (requires GITHUB_TOKEN)
release:
	goreleaser release --clean

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/

## help: print this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
