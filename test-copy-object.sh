#!/bin/bash
# Test script for CopyObject functionality

set -e

echo "Testing CopyObject functionality..."

# Start the server in local mode in the background
echo "Starting server in local mode..."
./tmp/main --local-repo /tmp/test-local-repo &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Function to cleanup
cleanup() {
  echo "Cleaning up..."
  kill $SERVER_PID 2>/dev/null || true
  wait $SERVER_PID 2>/dev/null || true
}
trap cleanup EXIT

# Test 1: Put an object
echo "1. Creating test object..."
echo "Hello, World!" | curl -X PUT \
  -H "Content-Type: text/plain" \
  --data-binary @- \
  "http://localhost:8080/test-bucket/hello.txt" || echo "Put failed"

# Test 2: Copy the object
echo "2. Copying object..."
curl -X PUT \
  -H "x-amz-copy-source: /test-bucket/hello.txt" \
  "http://localhost:8080/test-bucket/hello-copy.txt" || echo "Copy failed"

# Test 3: Verify the copy exists
echo "3. Verifying copied object..."
RESPONSE=$(curl -s "http://localhost:8080/test-bucket/hello-copy.txt" || echo "Get failed")
echo "Copied content: $RESPONSE"

# Test 4: Verify original still exists
echo "4. Verifying original object..."
ORIGINAL=$(curl -s "http://localhost:8080/test-bucket/hello.txt" || echo "Get original failed")
echo "Original content: $ORIGINAL"

echo "CopyObject test completed!"
