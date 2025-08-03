# Makefile for tiktok-whisper

# Variables
BINARY_NAME=v2t
GO_FILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")
CGO_ENABLED=1

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) go build -o $(BINARY_NAME) ./cmd/v2t/main.go

# Run unit tests only (with -short flag to skip integration tests)
.PHONY: test
test:
	go test -short -v ./...

# Run unit tests only (alias)
.PHONY: test-unit
test-unit: test

# Run integration tests
.PHONY: test-integration
test-integration: build
	@echo "Running integration tests..."
	@chmod +x scripts/test/integration_test.sh
	@./scripts/test/integration_test.sh

# Run Go integration tests
.PHONY: test-integration-go
test-integration-go: build
	CGO_ENABLED=$(CGO_ENABLED) go test -v -tags=integration ./...

# Run all tests
.PHONY: test-all
test-all: test test-integration-go

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f v2t_test
	rm -rf test/integration_*
	go clean

# Generate wire dependencies
.PHONY: wire
wire:
	cd internal/app && wire

# Format code
.PHONY: fmt
fmt:
	go fmt ./...
	gofmt -w $(GO_FILES)

# Run linter
.PHONY: lint
lint:
	golangci-lint run

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Database migration
.PHONY: db-migrate
db-migrate:
	@echo "Running database migration..."
	@cd scripts/migration && ./02_execute_migration.sh

# Database migration check
.PHONY: db-check
db-check:
	@echo "Checking database status..."
	@cd scripts/migration && ./03_post_migration_check.sh

# Run development server
.PHONY: run-web
run-web: build
	./$(BINARY_NAME) web --port :8081

# Quick test - run a single conversion
.PHONY: test-convert
test-convert: build
	@echo "Testing single file conversion..."
	@if [ -f "test/data/test.mp3" ]; then \
		./$(BINARY_NAME) convert single -a -i test/data/test.mp3 -u test_user; \
	else \
		echo "Test file not found: test/data/test.mp3"; \
	fi

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary"
	@echo "  make test           - Run unit tests only (fast, no external deps)"
	@echo "  make test-unit      - Alias for 'make test'"
	@echo "  make test-integration - Run shell integration tests"
	@echo "  make test-integration-go - Run Go integration tests"
	@echo "  make test-all       - Run all tests"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make wire           - Generate wire dependencies"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter"
	@echo "  make deps           - Install dependencies"
	@echo "  make db-migrate     - Run database migration"
	@echo "  make db-check       - Check database status"
	@echo "  make run-web        - Run web server"
	@echo "  make test-convert   - Test single file conversion"
	@echo "  make help           - Show this help"

.DEFAULT_GOAL := help