#!/bin/bash

# github-as-s3 Custom S3 API Test Runner
# Tests only the S3 operations that are actually supported

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Configuration
SERVER_PORT="${GHS3_PORT:-8080}"
SERVER_HOST="${GHS3_HOST:-localhost}"
ACCESS_KEY="${AWS_ACCESS_KEY_ID:-minioadmin}"
SECRET_KEY="${AWS_SECRET_ACCESS_KEY:-minioadmin}"
ENDPOINT="http://${SERVER_HOST}:${SERVER_PORT}"

echo "🧪 github-as-s3 Custom S3 Compatibility Test Runner"
echo "===================================================="
echo ""

# Check prerequisites
check_prerequisites() {
    echo "🔍 Checking prerequisites..."

    # Check if Go is installed
    if ! command -v go >/dev/null 2>&1; then
        echo "❌ Go is not installed"
        exit 1
    fi

    # Check required environment variables for remote mode
    if [ -z "$GHS3_LOCAL_REPO_PATH" ]; then
        if [ -z "$GITHUB_TOKEN" ]; then
            echo "❌ GITHUB_TOKEN environment variable is required (unless using local mode)"
            exit 1
        fi

        if [ -z "$GITHUB_OWNER" ]; then
            echo "❌ GITHUB_OWNER environment variable is required (unless using local mode)"
            exit 1
        fi
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
    echo "   Server will be available at ${ENDPOINT}"

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
        if curl -s --max-time 2 "${ENDPOINT}" >/dev/null 2>&1; then
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
    echo "🧪 Running S3 compatibility tests..."
    echo "   Endpoint: ${ENDPOINT}"
    echo "   Access Key: ${ACCESS_KEY}"
    echo ""

    # Set environment variables for the test
    export GHS3_ENDPOINT="${ENDPOINT}"
    export AWS_ACCESS_KEY_ID="${ACCESS_KEY}"
    export AWS_SECRET_ACCESS_KEY="${SECRET_KEY}"
    export AWS_REGION="us-east-1"

    # Initialize go module if needed
    cd test
    if [ ! -f go.sum ]; then
        echo "📦 Initializing test dependencies..."
        go mod tidy
        echo ""
    fi

    # Run the test suite
    go run s3_compatibility.go
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
    echo "   Check the logs in ./test/logs/ directory for detailed server logs"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
    --help)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Environment variables:"
        echo "  GITHUB_TOKEN         Required: GitHub API token (unless using local mode)"
        echo "  GITHUB_OWNER         Required: GitHub repository owner (unless using local mode)"
        echo "  GHS3_LOCAL_REPO_PATH Optional: Use local repository mode"
        echo "  GHS3_PORT            Optional: Server port (default: 8080)"
        echo "  GHS3_HOST            Optional: Server host (default: localhost)"
        echo "  AWS_ACCESS_KEY_ID    Optional: S3 access key (default: minioadmin)"
        echo "  AWS_SECRET_ACCESS_KEY Optional: S3 secret key (default: minioadmin)"
        echo ""
        echo "Examples:"
        echo "  # Basic usage with GitHub"
        echo "  export GITHUB_TOKEN=your_token"
        echo "  export GITHUB_OWNER=your_username"
        echo "  $0"
        echo ""
        echo "  # Run with local repository"
        echo "  export GHS3_LOCAL_REPO_PATH=/path/to/local/repo"
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
