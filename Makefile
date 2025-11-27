.PHONY: build clean test run install help fmt lint server run-server

# Variables
BINARY_NAME=registry-sync
SERVER_NAME=registry-sync-server
BUILD_DIR=.
CMD_DIR=cmd/cli
SERVER_DIR=cmd/server
GO=go
GOFLAGS=-v
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: build

## help: Display this help message
help:
	@echo "Available targets:"
	@echo "  build       - Build the CLI binary"
	@echo "  server      - Build the web server binary"
	@echo "  clean       - Remove built binaries and artifacts"
	@echo "  test        - Run tests"
	@echo "  run         - Run the CLI with example config"
	@echo "  run-server  - Run the web server"
	@echo "  install     - Install the binary to GOPATH/bin"
	@echo "  fmt         - Format code"
	@echo "  lint        - Run linters"
	@echo "  deps        - Download dependencies"

## build: Build the binary
build: deps
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## clean: Remove built binaries and artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	$(GO) clean
	@echo "✅ Clean complete"

## test: Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

## run: Run the application with example config
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) --config configs/sync.yaml --dry-run

## install: Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) $(CMD_DIR)/main.go
	@echo "✅ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "✅ Format complete"

## lint: Run linters (requires golangci-lint)
lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...
	@echo "✅ Lint complete"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✅ Dependencies downloaded"

## validate: Validate configuration
validate: build
	@echo "Validating configuration..."
	./$(BINARY_NAME) --config configs/sync.yaml --validate

## version: Display version
version:
	@echo "Version: $(VERSION)"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t registry-sync:$(VERSION) .
	@echo "✅ Docker image built: registry-sync:$(VERSION)"

## server: Build web server binary
server: deps
	@echo "Building $(SERVER_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_NAME) $(SERVER_DIR)/main.go
	@echo "✅ Server build complete: $(BUILD_DIR)/$(SERVER_NAME)"

## run-server: Run the web server
run-server: server
	@echo "Starting web server..."
	./$(SERVER_NAME)
