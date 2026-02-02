.PHONY: build build-web run clean test docker

VERSION ?= dev
BINARY_NAME = nanobot
BUILD_DIR = bin

# Default target
all: build

# Build the web UI
build-web:
	@echo "Building web UI..."
	cd web && npm install && npm run build

# Build the Go binary (development)
build:
	@echo "Building nanobot..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/nanobot

# Build with embedded web UI (production)
build-prod: build-web
	@echo "Building nanobot with embedded UI..."
	@mkdir -p $(BUILD_DIR)
	go build -tags=embed -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/nanobot

# Run in development mode
run:
	go run ./cmd/nanobot

# Run with hot reload (requires air)
dev:
	which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf web/dist
	go clean

# Run tests
test:
	go test -v ./...

# Build Docker image
docker:
	docker build -t nanobot:$(VERSION) .

# Release binaries for multiple platforms
release:
	@mkdir -p $(BUILD_DIR)/release
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 ./cmd/nanobot
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-arm64 ./cmd/nanobot
	# Darwin AMD64
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 ./cmd/nanobot
	# Darwin ARM64
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 ./cmd/nanobot
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe ./cmd/nanobot
	@echo "Release binaries built in $(BUILD_DIR)/release/"

# Install locally
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Format code
fmt:
	go fmt ./...
	gofmt -w .

# Lint
lint:
	which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

# Generate mocks (if needed)
generate:
	go generate ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

.DEFAULT_GOAL := build
