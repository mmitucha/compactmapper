.PHONY: all build build-windows build-macos test test-integration test-unit clean install run help

# Variables
APP_NAME = compactmapper
VERSION = 0.1.0
BUILD_DIR = build
LDFLAGS = -X main.version=$(VERSION)

# Default target
all: test build

# Build targets
build: ## Build compactmapper binary for current platform (GUI + CLI)
	@echo "Building $(APP_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(eval GOOS := $(shell go env GOOS))
	$(eval GOARCH := $(shell go env GOARCH))
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-$(GOOS)-$(GOARCH) ./cmd/compactmapper
	@ln -sf $(APP_NAME)-v$(VERSION)-$(GOOS)-$(GOARCH) $(BUILD_DIR)/$(APP_NAME)
	@echo "✓ Build complete: $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-$(GOOS)-$(GOARCH)"

# build-linux: Linux cross-compilation from macOS not supported
# Cross-compiling Fyne GUI apps for Linux requires complete Linux development environment:
# - X11 development headers (libX11-dev, libXcursor-dev, libXrandr-dev, libXinerama-dev, libXi-dev)
# - OpenGL development headers (libGL-dev, libGLU-dev)
# - pkg-config and other build tools
#
# Solutions:
# 1. Build natively on Linux: Run 'make build' on a Linux machine
# 2. Use GitHub Actions: Linux builds automated in CI/CD pipeline
# 3. Use Docker/fyne-cross: Requires Docker with complete Linux toolchain
#
# For GitHub Actions setup, see: .github/workflows/build.yml

build-windows: ## Cross-compile for Windows (requires mingw-w64)
	@echo "Building for Windows v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
		CC=x86_64-w64-mingw32-gcc \
		CXX=x86_64-w64-mingw32-g++ \
		CGO_CFLAGS="-O2 -g" \
		CGO_LDFLAGS="-O2 -g" \
		go build -ldflags '-H windowsgui -X main.version=$(VERSION)' \
		-o $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-windows-amd64.exe ./cmd/compactmapper
	@echo "✓ Build complete: $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-windows-amd64.exe"

build-macos: ## Build for macOS (native build)
	@echo "Building for macOS v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		CGO_CFLAGS="-O2 -g" \
		CGO_LDFLAGS="-O2 -g" \
		go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-darwin-amd64 ./cmd/compactmapper
	@echo "✓ Build complete: $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-darwin-amd64"

# Test targets
test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests only (fast)
	@echo "Running unit tests..."
	@go test -v ./internal/sorter ./internal/converter ./las

test-integration: ## Run component integration tests (Go tests with build tag)
	@echo "Running component integration tests..."
	@go test -v -tags=integration -timeout 60s ./test

test-e2e: clean build ## Run end-to-end black-box tests (bash script)
	@echo "Running end-to-end integration tests..."
	@./test/integration_test.sh

test-all: test-unit test-integration test-e2e ## Run all test levels (unit + integration + e2e)

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# Development targets
run: build ## Build and run the GUI
	$(BUILD_DIR)/$(APP_NAME)

run-cli: build ## Build and run CLI with test data
	$(BUILD_DIR)/$(APP_NAME) --input testdata/integration/input/data.csv --output ./tmp

clean: ## Clean build artifacts and test outputs
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html tmp/
	@echo "✓ Clean complete"

install: build ## Install to /usr/local/bin (requires sudo on macOS/Linux)
	@echo "Installing $(APP_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "✓ Installation complete"

# Utility targets
fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Format complete"

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@golangci-lint run
	@echo "✓ Lint complete"

deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

# Quick manual test
test-sample: build ## Quick manual test with test data
	@echo "Testing with sample data..."
	@mkdir -p tmp
	$(BUILD_DIR)/$(APP_NAME) --input testdata/integration/input/data.csv --output ./tmp
	@echo "✓ Sample test complete"
	@echo "  CSV output: ./tmp/csv"
	@echo "  LAS output: ./tmp/las"

# Help target (default when running 'make help')
help: ## Show this help message
	@echo "CompactMapper - Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
