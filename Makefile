.PHONY: help build run clean demo test test-coverage test-coverage-html test-verbose build-server run-server server version version-show version-set version-bump-major version-bump-minor version-bump-patch version-tag

# Variables
BINARY_NAME=demo
DEMO_DIR=cmd/demo
SERVER_BINARY_NAME=achemdb-server
SERVER_DIR=cmd/achemdb-server
BUILD_DIR=bin
COVERAGE_DIR=coverage

# Default target
help:
	@echo "Available targets:"
	@echo "  make build              - Build the demo binary"
	@echo "  make run                - Run the demo directly (without building)"
	@echo "  make demo                - Build and run the demo"
	@echo "  make build-server        - Build the achemdb-server binary"
	@echo "  make run-server          - Run the achemdb-server directly (without building)"
	@echo "  make server              - Build and run the achemdb-server"
	@echo "  make test                - Run all tests"
	@echo "  make test-verbose        - Run tests with verbose output"
	@echo "  make test-coverage       - Run tests with coverage report"
	@echo "  make test-coverage-html  - Generate HTML coverage report"
	@echo "  make clean               - Remove build artifacts"
	@echo ""
	@echo "Version management:"
	@echo "  make version             - Show current version"
	@echo "  make version-set VERSION=x.y.z  - Set version to x.y.z"
	@echo "  make version-bump-major  - Bump major version (x.0.0)"
	@echo "  make version-bump-minor  - Bump minor version (x.y.0)"
	@echo "  make version-bump-patch  - Bump patch version (x.y.z)"
	@echo "  make version-tag         - Create git tag for current version"

# Build the demo binary
build:
	@echo "Building demo..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(DEMO_DIR)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the demo directly (without building)
run:
	@echo "Running demo..."
	@go run ./$(DEMO_DIR)

# Build and run the demo
demo: build
	@echo "Running demo..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Build the achemdb-server binary
build-server:
	@echo "Building achemdb-server..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(SERVER_BINARY_NAME) ./$(SERVER_DIR)
	@echo "Binary built: $(BUILD_DIR)/$(SERVER_BINARY_NAME)"

# Run the achemdb-server directly (without building)
run-server:
	@echo "Running achemdb-server..."
	@go run ./$(SERVER_DIR)

# Build and run the achemdb-server
server: build-server
	@echo "Running achemdb-server..."
	@./$(BUILD_DIR)/$(SERVER_BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo ""
	@echo "Coverage report saved to $(COVERAGE_DIR)/coverage.out"

# Generate HTML coverage report
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "HTML coverage report saved to $(COVERAGE_DIR)/coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	@echo "Clean complete"

# Version management targets
VERSION_SCRIPT=scripts/version.sh

# Show current version
version:
	@$(VERSION_SCRIPT) show

# Set version (requires VERSION variable)
# Usage: make version-set VERSION=1.0.0
version-set:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION variable is required"; \
		echo "Usage: make version-set VERSION=x.y.z"; \
		exit 1; \
	fi
	@$(VERSION_SCRIPT) set $(VERSION)

# Bump major version
version-bump-major:
	@$(VERSION_SCRIPT) bump major

# Bump minor version
version-bump-minor:
	@$(VERSION_SCRIPT) bump minor

# Bump patch version
version-bump-patch:
	@$(VERSION_SCRIPT) bump patch

# Create git tag for current version
version-tag:
	@$(VERSION_SCRIPT) tag

