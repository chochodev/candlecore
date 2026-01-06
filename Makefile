# Candlecore Makefile

.PHONY: build run test clean fmt vet lint help

# Build the application
build:
	go build -o candlecore.exe ./cmd/candlecore

# Run the application
run: build
	./candlecore.exe

# Run with debug logging
run-debug: build
	$env:CANDLECORE_LOG_LEVEL="debug"; ./candlecore.exe

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Clean build artifacts and state
clean:
	Remove-Item -Force -ErrorAction SilentlyContinue candlecore.exe
	Remove-Item -Recurse -Force -ErrorAction SilentlyContinue .state

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Show help
help:
	@echo "Candlecore Makefile Commands:"
	@echo "  make build         - Build the application"
	@echo "  make run           - Build and run the application"
	@echo "  make run-debug     - Run with debug logging"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make clean         - Clean build artifacts and state"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make deps          - Download and tidy dependencies"
	@echo "  make help          - Show this help message"
