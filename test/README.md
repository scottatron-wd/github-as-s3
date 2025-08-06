# S3 API Testing with MinIO Mint

This directory contains tools and scripts for testing the S3 API compatibility of github-as-s3 using [MinIO Mint](https://github.com/minio/mint), a comprehensive S3 compatibility testing framework.

## Overview

MinIO Mint is the gold standard for testing S3 API compatibility. It includes tests for:

- AWS CLI (awscli)
- AWS SDKs (Go, Java, JavaScript, Python, PHP, Ruby)
- MinIO clients (Go, Java, JavaScript, Python)
- S3 tools (s3cmd, mc)
- Advanced features (versioning, select, health checks)

## Quick Start

### Prerequisites

1. **Required Environment Variables:**

   ```bash
   export GITHUB_TOKEN="your_github_token"
   export GITHUB_OWNER="your_github_username"
   ```

2. **Optional Environment Variables:**

   ```bash
   export GHS3_LOCAL_REPO_PATH="/path/to/local/repo"  # For local mode
   export MINT_MODE="full"  # core (default) or full
   export GHS3_PORT="8080"  # Server port (default: 8080)
   ```

3. **Container Runtime:**
   - Docker or Podman must be installed
   - MinIO Mint runs as a container image

### Running Tests

#### Option 1: Using the Test Runner (Recommended)

```bash
# Run core S3 compatibility tests
make test-s3

# Run full S3 compatibility tests  
make test-full

# Or run directly
./test/run-tests.sh
```

#### Option 2: Using Docker Compose

```bash
# Set environment variables first
export GITHUB_TOKEN="your_token"
export GITHUB_OWNER="your_owner"

# Run with Docker Compose
make test-docker
```

#### Option 3: Manual Setup

```bash
# 1. Start github-as-s3 server
go run ./cmd/cli/ &

# 2. Run MinIO Mint tests
./test/mint-test.sh --host localhost --port 8080
```

## Test Scripts

### `run-tests.sh`

Complete test automation script that:

- Builds the github-as-s3 application
- Starts the server in background
- Runs MinIO Mint tests
- Collects and summarizes results
- Cleans up automatically

### `mint-test.sh`

MinIO Mint wrapper script that:

- Configures test parameters
- Runs the MinIO Mint container
- Extracts and analyzes test results
- Provides detailed failure reports

## Test Results

### Log Files

Test results are saved in timestamped directories:

```
test/
├── logs/                    # Server logs
│   └── server-YYYYMMDD-HHMMSS.log
└── mint-logs-YYYYMMDD-HHMMSS/  # Test results
    ├── log.json            # Detailed test results
    └── other-mint-logs
```

### Result Format

Each test result in `log.json` contains:

```json
{
  "name": "aws-sdk-go",
  "function": "PutObject",
  "args": {"Bucket": "test-bucket", "Key": "test-key"},
  "duration": 245,
  "status": "PASS",
  "alert": "",
  "message": "Object uploaded successfully",
  "error": ""
}
```

Status values:

- `PASS`: Test passed
- `FAIL`: Test failed (implementation issue)
- `NA`: Test not applicable (feature not supported)

## Interpreting Results

### Expected Failures

Some tests may fail due to unimplemented S3 features:

- Advanced bucket policies
- Cross-origin resource sharing (CORS)
- Server-side encryption
- Multipart upload edge cases
- S3 Select queries

### Critical Tests

Focus on these core S3 operations:

- ✅ **ListBuckets**: Essential for S3 compatibility
- ✅ **CreateBucket**: Basic bucket operations
- ✅ **PutObject**: File upload functionality
- ✅ **GetObject**: File download functionality
- ✅ **DeleteObject**: File deletion
- ✅ **CopyObject**: File copying (recently implemented)
- ⚠️ **ListObjects**: Directory listing (partial support)

### Performance Metrics

Monitor test durations to identify performance bottlenecks:

- Fast operations: < 100ms (metadata operations)
- Medium operations: 100-1000ms (small file operations)
- Slow operations: > 1000ms (large file operations, git operations)

## Test Modes

### Core Mode (Default)

Tests essential S3 operations:

- Basic object operations (PUT, GET, DELETE, COPY)
- Bucket operations (CREATE, LIST, DELETE)
- Metadata operations
- Error handling

**Runtime:** 2-5 minutes

### Full Mode

Includes all core tests plus:

- Advanced S3 features
- Edge cases and error conditions
- Performance and stress tests
- SDK-specific functionality

**Runtime:** 10-30 minutes

## Troubleshooting

### Common Issues

#### Server Not Starting

```bash
# Check if port is already in use
lsof -i :8080

# Check server logs
tail -f test/logs/server-*.log
```

#### Container Runtime Issues

```bash
# For Docker
docker --version
docker run hello-world

# For Podman
podman --version
podman run hello-world
```

#### GitHub API Issues

```bash
# Test GitHub token
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user

# Check rate limits
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/rate_limit
```

### Test Failures Analysis

#### Authentication Failures

- Check AWS credentials configuration
- Verify S3 authentication implementation

#### Network/Connectivity Issues

- Ensure server is accessible from container
- Check firewall and network policies

#### Implementation Gaps

- Review failed test details in log.json
- Prioritize based on S3 operation importance
- Consider partial implementation for complex features

## Configuration Options

### Server Configuration

```bash
# Change server port
export GHS3_PORT=9000

# Enable local repository mode
export GHS3_LOCAL_REPO_PATH="/path/to/repo"

# Configure GitHub settings
export GITHUB_OWNER="organization-name"
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
```

### Test Configuration

```bash
# Test mode selection
export MINT_MODE="core"    # Basic tests (default)
export MINT_MODE="full"    # Comprehensive tests

# Custom access credentials
export AWS_ACCESS_KEY_ID="custom-key"
export AWS_SECRET_ACCESS_KEY="custom-secret"
```

## Contributing

### Adding New Tests

1. Follow MinIO Mint structure in `mint/build/` and `mint/run/core/`
2. Create test scripts following existing patterns
3. Update documentation with new test coverage

### Improving Test Scripts

1. Enhance error handling and reporting
2. Add more detailed result analysis
3. Optimize test performance and reliability

### Reporting Issues

When reporting test failures:

1. Include full test logs (`log.json`)
2. Specify github-as-s3 version and configuration
3. Provide minimal reproduction steps
4. Include server logs if relevant

## References

- [MinIO Mint Documentation](https://github.com/minio/mint)
- [AWS S3 API Reference](https://docs.aws.amazon.com/s3/latest/API/)
- [S3 Compatibility Guidelines](https://min.io/docs/minio/linux/developers/s3-compatible-api.html)
