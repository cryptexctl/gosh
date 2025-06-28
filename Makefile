.PHONY: build install clean test run fmt vet lint help

BINARY_NAME=gosh
VERSION?=1.0.0
BUILD_TIME=$(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

help: ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

install: build ## Install to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	go clean

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

run: build ## Build and run
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint: fmt vet ## Run linters

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

release: clean lint test build ## Create release build
	@echo "Creating release build..."
	mkdir -p dist
	cp $(BINARY_NAME) dist/
	cp README.md dist/
	cd dist && tar -czf $(BINARY_NAME)-$(VERSION)-$(shell uname -s)-$(shell uname -m).tar.gz $(BINARY_NAME) README.md

dev: ## Development build with debug info
	@echo "Building development version..."
	go build -race $(LDFLAGS) -o $(BINARY_NAME) .

check: fmt vet ## Quick check (format + vet)
	@echo "Quick check completed"

all: clean lint test build ## Clean, lint, test and build 