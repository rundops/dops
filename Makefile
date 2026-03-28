BINARY := dops
VERSION := 0.1.0
BUILD_DIR := bin
LDFLAGS := -ldflags "-s -w -X dops/cmd.version=$(VERSION)"

.PHONY: all build test lint clean install screenshots docker web web-dev

## Build

all: build

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

build-all:
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-arm64.exe .

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

## Test

test:
	go test ./... -v -timeout 60s

test-short:
	go test ./... -timeout 60s

test-race:
	go test ./... -race -timeout 60s

test-cover:
	go test ./... -coverprofile=coverage.out -timeout 60s
	go tool cover -html=coverage.out -o coverage.html

## Lint

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

## Screenshots (VHS)

screenshots: build
	@echo "Generating README hero screenshots and demo GIF..."
	vhs tapes/readme-hero.tape
	@echo "Done. Assets in assets/"

tapes: build
	@echo "Generating feature screenshots..."
	@for tape in tapes/demo-*.tape; do \
		echo "  Running $$tape..."; \
		vhs $$tape; \
	done
	@echo "Done. Screenshots in tapes/screenshots/, GIFs in tapes/gifs/"

## Web UI

web:
	cd web && npm ci && npm run build

web-dev:
	cd web && npm run dev

## Docker

docker:
	docker build -t $(BINARY):$(VERSION) .

docker-run:
	docker run -i -v ~/.dops:/data/dops $(BINARY):$(VERSION)

docker-run-http:
	docker run -p 8080:8080 -v ~/.dops:/data/dops $(BINARY):$(VERSION) --transport http --port 8080

## Release (local)

release-local:
	goreleaser release --snapshot --clean

## Clean

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

## CI

ci: vet test-short build
