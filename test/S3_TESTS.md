# Custom S3 API Test Suite

This directory contains a custom S3 API compatibility test suite specifically designed for github-as-s3. Unlike MinIO Mint which tests the complete S3 API and fails early on missing features, this test suite only tests the operations that github-as-s3 actually supports.

## Supported S3 Operations Tested

✅ **Bucket Operations**

- `ListBuckets` - List all available buckets
- `CreateBucket` - Create a new bucket (repository)
- `HeadBucket` - Check if bucket exists
- `DeleteBucket` - Delete a bucket (repository)

✅ **Object Operations**

- `PutObject` - Upload objects (files)
- `GetObject` - Download objects (files)
- `HeadObject` - Get object metadata
- `DeleteObject` - Delete objects (files)
- `CopyObject` - Copy objects within the same bucket
- `ListObjectsV2` - List objects in bucket with prefix support

✅ **Advanced Tests**

- Multiple object operations
- Prefix-based object listing
- Object content verification
- Error handling for deleted objects

## Quick Start

### Prerequisites

1. **For GitHub mode (default):**

   ```bash
   export GITHUB_TOKEN="your_github_token"
   export GITHUB_OWNER="your_github_username"
   ```

2. **For Local mode:**

   ```bash
   export GHS3_LOCAL_REPO_PATH="/path/to/your/git/repo"
   ```

### Running Tests

```bash
# Using the Makefile (recommended)
make test-s3-custom

# Or run directly
./test/run-s3-tests.sh

# Manual execution
cd test
go run s3_compatibility_test.go
```

## Test Configuration

The test suite can be configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `GHS3_ENDPOINT` | `http://localhost:8080` | github-as-s3 server endpoint |
| `AWS_ACCESS_KEY_ID` | `minioadmin` | S3 access key |
| `AWS_SECRET_ACCESS_KEY` | `minioadmin` | S3 secret key |
| `AWS_REGION` | `us-east-1` | AWS region |
| `GHS3_PORT` | `8080` | Server port |
| `GHS3_HOST` | `localhost` | Server host |

## Test Results

The test suite provides detailed output including:

- ✅ Real-time test execution with pass/fail status
- ⏱️ Execution time for each test
- 📊 Summary with success rate
- ❌ Detailed error information for failed tests
- 🧹 Automatic cleanup of test resources

Example output:

```
🧪 github-as-s3 S3 API Compatibility Test Suite
================================================
Endpoint: http://localhost:8080
Access Key: minioadmin

🚀 Running S3 API tests...

✅ ListBuckets (45ms)
✅ CreateBucket (123ms)
✅ HeadBucket (34ms)
✅ PutObject (167ms)
✅ GetObject (89ms)
✅ HeadObject (45ms)
✅ ListObjectsV2 (78ms)
✅ CopyObject (134ms)
✅ GetCopiedObject (67ms)
✅ DeleteObject (98ms)
✅ DeleteCopiedObject (87ms)
✅ GetDeletedObject (23ms)
✅ ListObjectsWithPrefix (234ms)
✅ MultipleObjectOperations (456ms)
✅ DeleteBucket (123ms)

📊 Test Results Summary
=======================
✅ Passed: 15
❌ Failed: 0
📊 Total: 15

🎯 Success Rate: 100.0%
```

## Test Details

### Basic Operations Flow

1. **ListBuckets** - Verify the server responds to bucket listing
2. **CreateBucket** - Create a test bucket (GitHub repository)
3. **HeadBucket** - Verify the bucket exists
4. **PutObject** - Upload a test file
5. **GetObject** - Download and verify the file content
6. **HeadObject** - Check object metadata
7. **ListObjectsV2** - Verify object appears in listing

### Copy Operations

8. **CopyObject** - Copy the test file to a new location
9. **GetCopiedObject** - Verify copied file has correct content

### Cleanup Operations

10. **DeleteObject** - Delete the original test file
11. **DeleteCopiedObject** - Delete the copied file
12. **GetDeletedObject** - Verify deletion (should return NoSuchKey)
13. **DeleteBucket** - Clean up the test bucket

### Advanced Operations

14. **ListObjectsWithPrefix** - Test prefix-based filtering
15. **MultipleObjectOperations** - Test handling multiple objects

## Advantages over MinIO Mint

✅ **Targeted Testing**: Only tests supported operations  
✅ **No Early Failures**: Continues testing even if some operations fail  
✅ **Custom Validation**: Specific checks for github-as-s3 behavior  
✅ **Faster Execution**: ~2-5 seconds vs 2-30 minutes for Mint  
✅ **Detailed Reporting**: Clear pass/fail status for each operation  
✅ **Easy Debugging**: Focused error messages for supported features  

## Integration with CI/CD

```yaml
# Example GitHub Actions workflow
- name: Run S3 Compatibility Tests
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    GITHUB_OWNER: ${{ github.repository_owner }}
  run: make test-s3-custom
```

## Extending the Test Suite

To add new tests:

1. Create a new test function following the pattern:

   ```go
   func TestNewFeature(config *TestConfig) error {
       // Test implementation
       return nil // or error
   }
   ```

2. Add it to the main execution in `s3_compatibility_test.go`:

   ```go
   ts.RunTest("NewFeature", TestNewFeature)
   ```

The test suite automatically handles:

- Error reporting and status tracking
- Execution timing
- Resource cleanup
- Result summarization

## Files

- `s3_compatibility_test.go` - Main test suite implementation
- `run-s3-tests.sh` - Test runner script with server management
- `go.mod` / `go.sum` - Go module dependencies
- `S3_TESTS.md` - This documentation file
