.PHONY: build build-web run clean test docker install install-local release

VERSION ?= dev
BINARY_NAME = goclawde
BUILD_DIR = bin
WEB_DIR = web

# Install directory (default: ~/.local/bin for user install, /usr/local/bin for system)
ifeq ($(USER),root)
    INSTALL_DIR = /usr/local/bin
else
    INSTALL_DIR = $(HOME)/.local/bin
endif

# Default target
all: build

# Build the web UI (just copy index.html for now)
build-web:
	@echo "Preparing web UI..."
	@mkdir -p $(WEB_DIR)/dist
	@cp $(WEB_DIR)/index.html $(WEB_DIR)/dist/
	@mkdir -p $(BUILD_DIR)/web
	@cp $(WEB_DIR)/index.html $(BUILD_DIR)/web/

# Build the Go binary (development)
build: build-web
	@echo "Building GoClawde..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/goclawde

# Build with embedded web UI (production)
build-prod: build-web
	@echo "Building GoClawde (production)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/goclawde

# Run in development mode
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run with hot reload (requires air)
dev:
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean

# Run tests
test:
	go test -v ./...

# Build Docker image
docker:
	docker build -t goclawde.ai:$(VERSION) .

# Release binaries for multiple platforms
release: clean
	@mkdir -p $(BUILD_DIR)/release
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 ./cmd/goclawde
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-arm64 ./cmd/goclawde
	# Darwin AMD64
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 ./cmd/goclawde
	# Darwin ARM64
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 ./cmd/goclawde
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -s -w" -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe ./cmd/goclawde
	@echo "Release binaries built in $(BUILD_DIR)/release/"

# Install locally to user's bin directory (no sudo needed)
install-local: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@echo "✓ Installed to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "Make sure $(INSTALL_DIR) is in your PATH:"
	@echo "  export PATH=\"\$$PATH:$(INSTALL_DIR)\""
	@echo ""
	@echo "Or add to your shell config (~/.bashrc, ~/.zshrc, etc.):"
	@echo '  export PATH="$$PATH:$(INSTALL_DIR)"'

# Install system-wide (requires sudo)
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✓ Installed to /usr/local/bin/$(BINARY_NAME)"

# Uninstall
uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✓ Uninstalled $(BINARY_NAME)"

# Format code
fmt:
	go fmt ./...

# Lint
lint:
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Setup development environment
setup:
	go mod init github.com/YOUR_USERNAME/goclawde.ai 2>/dev/null || true
	go mod tidy

.DEFAULT_GOAL := build
