.PHONY: help build run clean demo test test-coverage test-coverage-html test-verbose

# Variables
BINARY_NAME=demo
DEMO_DIR=cmd/demo
BUILD_DIR=bin
COVERAGE_DIR=coverage

# Default target
help:
	@echo "Available targets:"
	@echo "  make build              - Build the demo binary"
	@echo "  make run                - Run the demo directly (without building)"
	@echo "  make demo                - Build and run the demo"
	@echo "  make test                - Run all tests"
	@echo "  make test-verbose        - Run tests with verbose output"
	@echo "  make test-coverage       - Run tests with coverage report"
	@echo "  make test-coverage-html  - Generate HTML coverage report"
	@echo "  make clean               - Remove build artifacts"

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

