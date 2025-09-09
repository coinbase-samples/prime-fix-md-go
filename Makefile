# Prime FIX MD Go - Makefile

.PHONY: build test test-verbose test-coverage clean run lint fmt

# Build the application
build:
	go build -o fix-md-client ./cmd

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests for specific package
test-database:
	go test -v ./database

test-fixclient:
	go test -v ./fixclient

test-formatter:
	go test -v ./formatter

test-integration:
	go test -v -run TestIntegration

# Clean build artifacts
clean:
	rm -f fix-md-client
	rm -f coverage.out
	rm -f coverage.html
	rm -f *.db
	rm -f *.db-shm
	rm -f *.db-wal

# Run the application
run: build
	./fix-md-client

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests in CI/CD environment
test-ci:
	go test -race -coverprofile=coverage.out ./...

# Quick development cycle
dev: fmt test build

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  test           - Run all tests"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-database  - Run database tests only"
	@echo "  test-fixclient - Run FIX client tests only"
	@echo "  test-formatter - Run formatter tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Build and run the application"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  deps           - Install dependencies"
	@echo "  test-ci        - Run tests for CI/CD"
	@echo "  dev            - Quick development cycle (fmt + test + build)"
	@echo "  help           - Show this help"