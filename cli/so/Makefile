# Makefile for the Socle project (located in cli/so)

# Variables
BINARY_NAME=so
BINARY_DIR=../bin # Output relative to cli/so, placing it in socle/bin
PKG_PATH=. # Build the package in the current directory

# Default target (optional, runs when you just type 'make')
default: build

# Build the application binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY_DIR)/$(BINARY_NAME) $(PKG_PATH)
	@echo "$(BINARY_NAME) built in $(BINARY_DIR)/"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code (Requires golangci-lint: https://golangci-lint.run/usage/install/)
lint:
	@echo "Linting code..."
	@golangci-lint run ./... || echo "Linting failed, please check errors."

# Lint and attempt to fix code automatically
lint-fix:
	@echo "Linting code and attempting auto-fix..."
	@golangci-lint run --fix ./...
	@echo "Lint fix attempt complete."

# Install the binary
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(PKG_PATH)
	@echo "$(BINARY_NAME) installed."

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_DIR)/$(BINARY_NAME)
	@echo "Clean complete."

# Use Go to generate the CLI reference documentation in cli/so/README.md
docs:
	@echo "Generating CLI documentation..."
	@go run ./internal/docgen/main.go
	@echo "CLI documentation updated in README.md"

# Run common checks (Example 'all' target)
all: fmt lint test

# Development build - installs so-dev globally
dev:
	go build -o $(shell go env GOPATH)/bin/so-dev ./

# Declare targets that are not files
.PHONY: default build dev-install test fmt lint lint-fix install clean docs all
