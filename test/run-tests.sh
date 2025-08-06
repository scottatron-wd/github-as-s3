#!/bin/bash

# Quick S3 API Test Runner for github-as-s3
# This script starts the server and runs basic S3 compatibility tests

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Configuration
SERVER_PORT="${GHS3_PORT:-8080}"
SERVER_HOST="${GHS3_HOST:-localhost}"
MINT_SERVER_HOST="host.docker.internal" # Use this for Docker on macOS/Windows
ACCESS_KEY="${AWS_ACCESS_KEY_ID:-minioadmin}"
SECRET_KEY="${AWS_SECRET_ACCESS_KEY:-minioadmin}"
TEST_MODE="${MINT_MODE:-core}"

echo "🧪 github-as-s3 S3 API Test Runner"
echo "=================================="
echo ""

# Check prerequisites
check_prerequisites() {
  echo "🔍 Checking prerequisites..."

  # Check if Go is installed
  if ! command -v go >/dev/null 2>&1; then
    echo "❌ Go is not installed"
    exit 1
  fi

  # Check if Docker/Podman is available
  if ! command -v docker >/dev/null 2>&1 && ! command -v podman >/dev/null 2>&1; then
    echo "❌ Neither Docker nor Podman is installed"
    echo "   MinIO Mint requires Docker or Podman to run"
    exit 1
  fi

  # Check required environment variables
  if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ GITHUB_TOKEN environment variable is required"
    exit 1
  fi

  if [ -z "$GITHUB_OWNER" ]; then
    echo "❌ GITHUB_OWNER environment variable is required"
    exit 1
  fi

  echo "✅ Prerequisites check passed"
  echo ""
}

# Build the project
build_project() {
  echo "🔨 Building github-as-s3..."
  go build -o ./bin/github-as-s3 ./cmd/cli/
  echo "✅ Build completed"
  echo ""
}

# Start the server in background
start_server() {
  echo "🚀 Starting github-as-s3 server..."
  echo "   Server will be available at http://${SERVER_HOST}:${SERVER_PORT}"

  # Create log directory
  mkdir -p ./test/logs

  # Start server in background
  ./bin/github-as-s3 >"./test/logs/server-$(date +%Y%m%d-%H%M%S).log" 2>&1 &
  SERVER_PID=$!

  echo "   Server started with PID: $SERVER_PID"
  echo "   Server logs: ./test/logs/server-$(date +%Y%m%d-%H%M%S).log"

  # Wait for server to be ready
  echo "   Waiting for server to be ready..."
  for i in {1..30}; do
    if curl -s --max-time 2 "http://${SERVER_HOST}:${SERVER_PORT}" >/dev/null 2>&1; then
      echo "✅ Server is ready"
      echo ""
      return
    fi
    sleep 1
  done

  echo "❌ Server failed to start or is not responding"
  kill $SERVER_PID 2>/dev/null || true
  exit 1
}

# Stop the server
stop_server() {
  if [ -n "$SERVER_PID" ]; then
    echo "🛑 Stopping server (PID: $SERVER_PID)..."
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    echo "✅ Server stopped"
  fi
}

# Run the tests
run_tests() {
  echo "🧪 Running MinIO Mint S3 compatibility tests..."
  echo "   Test mode: $TEST_MODE"
  echo "   This may take several minutes..."
  echo ""

  # Run the mint test script
  ./test/mint-test.sh \
    --host "$MINT_SERVER_HOST" \
    --port "$SERVER_PORT" \
    --access-key "$ACCESS_KEY" \
    --secret-key "$SECRET_KEY" \
    --mode "$TEST_MODE"
}

# Cleanup on exit
cleanup() {
  echo ""
  echo "🧹 Cleaning up..."
  stop_server
}

trap cleanup EXIT

# Main execution
main() {
  check_prerequisites
  build_project
  start_server
  run_tests

  echo ""
  echo "🎉 Test run completed!"
  echo "   Check the logs in ./test/ directory for detailed results"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
  --help)
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Environment variables:"
    echo "  GITHUB_TOKEN         Required: GitHub API token"
    echo "  GITHUB_OWNER         Required: GitHub repository owner"
    echo "  GHS3_LOCAL_REPO_PATH Optional: Use local repository mode"
    echo "  GHS3_PORT            Optional: Server port (default: 8080)"
    echo "  GHS3_HOST            Optional: Server host (default: localhost)"
    echo "  AWS_ACCESS_KEY_ID    Optional: S3 access key (default: minioadmin)"
    echo "  AWS_SECRET_ACCESS_KEY Optional: S3 secret key (default: minioadmin)"
    echo "  MINT_MODE            Optional: Test mode - core or full (default: core)"
    echo ""
    echo "Examples:"
    echo "  # Basic usage"
    echo "  export GITHUB_TOKEN=your_token"
    echo "  export GITHUB_OWNER=your_username"
    echo "  $0"
    echo ""
    echo "  # Run with local repository"
    echo "  export GHS3_LOCAL_REPO_PATH=/path/to/local/repo"
    echo "  $0"
    echo ""
    echo "  # Run full test suite"
    echo "  export MINT_MODE=full"
    echo "  $0"
    exit 0
    ;;
  *)
    echo "Unknown option: $1"
    echo "Use --help for usage information"
    exit 1
    ;;
  esac
done

# Run main function
main
