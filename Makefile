# github-as-s3 Makefile

.PHONY: build run test test-s3 test-full clean help mint-setup

# Default target
.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME := github-as-s3
BUILD_DIR := ./bin
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './mint/*')

# Test configuration
TEST_PORT := 8080
TEST_HOST := localhost

## Build the application
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/cli/
	@echo "✅ Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

## Run the application locally
run: build
	@echo "🚀 Starting $(BINARY_NAME)..."
	@echo "Make sure GITHUB_TOKEN and GITHUB_OWNER are set"
	$(BUILD_DIR)/$(BINARY_NAME)

## Run standard Go tests
test:
	@echo "🧪 Running Go tests..."
	go test -v ./...

## Run custom S3 API compatibility tests (recommended)
test-s3-custom: build
	@echo "🧪 Running custom S3 API compatibility tests..."
	@echo "Prerequisites: Set GITHUB_TOKEN and GITHUB_OWNER (or GHS3_LOCAL_REPO_PATH for local mode)"
	./test/run-s3-tests.sh

## Run S3 API compatibility tests (requires MinIO Mint)
test-s3: build mint-setup
	@echo "🧪 Running S3 API compatibility tests..."
	@echo "Prerequisites: GITHUB_TOKEN and GITHUB_OWNER must be set"
	./test/run-tests.sh

## Run full S3 API compatibility tests
test-full: build mint-setup
	@echo "🧪 Running full S3 API compatibility tests..."
	@echo "Prerequisites: GITHUB_TOKEN and GITHUB_OWNER must be set"
	MINT_MODE=full ./test/run-tests.sh

## Set up MinIO Mint for S3 testing (one-time setup)
mint-setup:
	@if [ ! -d "./mint" ]; then \
		echo "📦 Setting up MinIO Mint for S3 testing..."; \
		git clone https://github.com/minio/mint.git; \
		echo "✅ MinIO Mint setup completed"; \
	else \
		echo "✅ MinIO Mint already set up"; \
	fi
	@if command -v docker >/dev/null 2>&1; then \
		echo "🐳 Pulling MinIO Mint Docker image..."; \
		docker pull minio/mint:latest; \
	elif command -v podman >/dev/null 2>&1; then \
		echo "🐳 Pulling MinIO Mint Podman image..."; \
		podman pull minio/mint:latest; \
	else \
		echo "⚠️  Warning: Neither Docker nor Podman found. MinIO Mint requires a container runtime."; \
	fi

## Test with Docker Compose
test-docker: build
	@echo "🐳 Running tests with Docker Compose..."
	@echo "Prerequisites: GITHUB_TOKEN and GITHUB_OWNER must be set"
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit --profile test

## Clean build artifacts and test logs
clean:
	@echo "🧹 Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -rf ./test/logs
	rm -rf ./test/mint-logs*
	@echo "✅ Cleanup completed"

## Clean everything including MinIO Mint
clean-all: clean
	@echo "🧹 Deep cleaning..."
	rm -rf ./mint
	@echo "✅ Deep cleanup completed"

## Show this help message
help:
	@echo "github-as-s3 - S3-compatible API for GitHub repositories"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m\033[0m\n"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "Environment Variables:"
	@echo "  GITHUB_TOKEN         Required: GitHub API token"
	@echo "  GITHUB_OWNER         Required: GitHub repository owner"
	@echo "  GHS3_LOCAL_REPO_PATH Optional: Use local repository mode"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build the application"
	@echo "  make test-s3-custom           # Run custom S3 compatibility tests (recommended)"
	@echo "  make test-s3                  # Run MinIO Mint S3 compatibility tests"
	@echo "  make test-full                # Run full S3 compatibility tests"
	@echo "  make test-docker              # Run tests with Docker Compose"
