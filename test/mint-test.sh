#!/bin/bash

# MinIO Mint S3 API Testing Script for github-as-s3
# This script sets up and runs comprehensive S3 compatibility tests

set -e

# Default values
GITHUB_AS_S3_HOST="localhost"
GITHUB_AS_S3_PORT="8080"
ACCESS_KEY="minioadmin"
SECRET_KEY="minioadmin"
MINT_MODE="core"
SERVER_LOG_FILE="server.log"
MINT_CONTAINER_NAME="mint-test-$(date +%s)"

usage() {
  echo "Usage: $0 [OPTIONS]"
  echo "Options:"
  echo "  -h, --host HOST      github-as-s3 server host (default: localhost)"
  echo "  -p, --port PORT      github-as-s3 server port (default: 8080)"
  echo "  -a, --access-key KEY Access key (default: minioadmin)"
  echo "  -s, --secret-key KEY Secret key (default: minioadmin)"
  echo "  -m, --mode MODE      Mint test mode: core or full (default: core)"
  echo "  --help               Show this help message"
  echo ""
  echo "Environment variables can also be used:"
  echo "  GITHUB_TOKEN         Required for github-as-s3"
  echo "  GITHUB_OWNER         Required for github-as-s3"
  echo "  GHS3_LOCAL_REPO_PATH Optional: Use local repository mode"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
  -h | --host)
    GITHUB_AS_S3_HOST="$2"
    shift 2
    ;;
  -p | --port)
    GITHUB_AS_S3_PORT="$2"
    shift 2
    ;;
  -a | --access-key)
    ACCESS_KEY="$2"
    shift 2
    ;;
  -s | --secret-key)
    SECRET_KEY="$2"
    shift 2
    ;;
  -m | --mode)
    MINT_MODE="$2"
    shift 2
    ;;
  --help)
    usage
    exit 0
    ;;
  *)
    echo "Unknown option: $1"
    usage
    exit 1
    ;;
  esac
done

echo "🚀 Starting MinIO Mint S3 API Tests for github-as-s3"
echo "Server: ${GITHUB_AS_S3_HOST}:${GITHUB_AS_S3_PORT}"
echo "Mode: ${MINT_MODE}"
echo "Access Key: ${ACCESS_KEY}"
echo ""

# Check if Docker/Podman is available
if command -v podman >/dev/null 2>&1; then
  CONTAINER_CMD="podman"
elif command -v docker >/dev/null 2>&1; then
  CONTAINER_CMD="docker"
else
  echo "❌ Error: Neither Docker nor Podman is installed"
  echo "Please install Docker or Podman to run MinIO Mint tests"
  exit 1
fi

echo "Using container runtime: ${CONTAINER_CMD}"

# Check if github-as-s3 server is running
echo "🔍 Checking if github-as-s3 server is accessible..."
if ! curl -s --max-time 5 "http://${GITHUB_AS_S3_HOST}:${GITHUB_AS_S3_PORT}" >/dev/null 2>&1; then
  echo "⚠️  Warning: github-as-s3 server is not accessible at ${GITHUB_AS_S3_HOST}:${GITHUB_AS_S3_PORT}"
  echo "   Make sure the server is running before proceeding."
  echo ""
  echo "   To start the server:"
  echo "   export GITHUB_TOKEN=your_token"
  echo "   export GITHUB_OWNER=your_owner"
  echo "   go run ./cmd/cli/"
  echo ""
  read -p "Continue anyway? (y/n): " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
fi

# Create log directory
LOG_DIR="./test/mint-logs-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$LOG_DIR"

echo "📋 Test configuration:"
echo "  Server endpoint: ${GITHUB_AS_S3_HOST}:${GITHUB_AS_S3_PORT}"
echo "  Access key: ${ACCESS_KEY}"
echo "  Secret key: ${SECRET_KEY}"
echo "  Mint mode: ${MINT_MODE}"
echo "  Log directory: ${LOG_DIR}"
echo ""

# Run MinIO Mint tests
echo "🧪 Running MinIO Mint tests..."
echo "This may take several minutes depending on the test mode..."

set +e # Don't exit on test failures

$CONTAINER_CMD run --rm \
  --name "$MINT_CONTAINER_NAME" \
  -e "SERVER_ENDPOINT=${GITHUB_AS_S3_HOST}:${GITHUB_AS_S3_PORT}" \
  -e "ACCESS_KEY=${ACCESS_KEY}" \
  -e "SECRET_KEY=${SECRET_KEY}" \
  -e "ENABLE_HTTPS=0" \
  -e "MINT_MODE=${MINT_MODE}" \
  -e "RUN_ON_FAIL=1" \
  -v "${PWD}/${LOG_DIR}:/mint/log" \
  minio/mint

TEST_EXIT_CODE=$?
set -e

echo ""
echo "📊 Test Results Summary"
echo "======================="

# Parse and display test results
if [ -f "${LOG_DIR}/log.json" ]; then
  echo "📄 Test log saved to: ${LOG_DIR}/log.json"
  echo ""

  # Count results by status
  PASS_COUNT=$(grep -o '"status":"PASS"' "${LOG_DIR}/log.json" | wc -l || echo 0)
  FAIL_COUNT=$(grep -o '"status":"FAIL"' "${LOG_DIR}/log.json" | wc -l || echo 0)
  NA_COUNT=$(grep -o '"status":"NA"' "${LOG_DIR}/log.json" | wc -l || echo 0)

  echo "✅ Passed: $PASS_COUNT"
  echo "❌ Failed: $FAIL_COUNT"
  echo "⏭️  N/A: $NA_COUNT"
  echo ""

  if [ "$FAIL_COUNT" -gt 0 ]; then
    echo "❌ Failed Tests:"
    echo "==============="
    # Extract failed test details
    grep '"status":"FAIL"' "${LOG_DIR}/log.json" |
      while IFS= read -r line; do
        name=$(echo "$line" | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
        function=$(echo "$line" | grep -o '"function":"[^"]*"' | cut -d'"' -f4)
        alert=$(echo "$line" | grep -o '"alert":"[^"]*"' | cut -d'"' -f4 || echo "")
        echo "  • $name: $function"
        if [ -n "$alert" ]; then
          echo "    Alert: $alert"
        fi
      done
    echo ""
  fi
else
  echo "⚠️  No test log file found at ${LOG_DIR}/log.json"
  echo "   Check if the tests ran successfully."
fi

echo "🔍 Full test results and logs are available in: ${LOG_DIR}/"
echo ""

if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo "🎉 All tests completed successfully!"
else
  echo "⚠️  Some tests failed or there were issues running the tests."
  echo "   Check the logs in ${LOG_DIR}/ for more details."
fi

echo ""
echo "💡 Tips:"
echo "  • Review the log.json file for detailed test results"
echo "  • Failed tests may indicate areas where S3 compatibility can be improved"
echo "  • Some tests may fail due to unimplemented S3 features"

exit $TEST_EXIT_CODE
