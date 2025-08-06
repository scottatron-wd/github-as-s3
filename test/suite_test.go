package s3test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestResult represents the result of a single test
type TestResult struct {
	Name     string
	Status   string
	Duration time.Duration
	Error    string
}

// TestSuite manages the S3 compatibility test execution
type TestSuite struct {
	Config  *TestConfig
	Results []TestResult
}

// TestConfig holds configuration for the test suite
type TestConfig struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	Region         string
	TestBucket     string
	S3Client       *s3.S3
	CleanupBuckets []string
}

// NewTestConfig creates a new test configuration
func NewTestConfig() *TestConfig {
	endpoint := getEnv("GHS3_ENDPOINT", "http://localhost:8080")
	accessKey := getEnv("AWS_ACCESS_KEY_ID", "minioadmin")
	secretKey := getEnv("AWS_SECRET_ACCESS_KEY", "minioadmin")
	region := getEnv("AWS_REGION", "us-east-1")

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(true),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create AWS session: %v", err))
	}

	return &TestConfig{
		Endpoint:   endpoint,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		Region:     region,
		TestBucket: fmt.Sprintf("test-bucket-%d", time.Now().Unix()),
		S3Client:   s3.New(sess),
	}
}

// NewTestSuite creates a new test suite
func NewTestSuite() *TestSuite {
	return &TestSuite{
		Config:  NewTestConfig(),
		Results: []TestResult{},
	}
}

// RunTest executes a single test function and records the result
func (ts *TestSuite) RunTest(name string, testFunc func(*TestConfig) error) {
	start := time.Now()
	fmt.Printf("  • %-25s ", name)

	err := testFunc(ts.Config)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ FAIL (%v) - %s\n", duration, err.Error())
		ts.Results = append(ts.Results, TestResult{
			Name:     name,
			Status:   "FAIL",
			Duration: duration,
			Error:    err.Error(),
		})
	} else {
		fmt.Printf("✅ PASS (%v)\n", duration)
		ts.Results = append(ts.Results, TestResult{
			Name:     name,
			Status:   "PASS",
			Duration: duration,
		})
	}
}

// Cleanup removes any test buckets that were created
func (ts *TestSuite) Cleanup() {
	for _, bucket := range ts.Config.CleanupBuckets {
		_ = deleteBucketRecursive(ts.Config, bucket)
	}
}

// PrintSummary prints a summary of all test results
func (ts *TestSuite) PrintSummary() {
	fmt.Println()
	fmt.Println("📊 Test Summary")
	fmt.Println("===============")

	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	for _, result := range ts.Results {
		totalDuration += result.Duration
		if result.Status == "PASS" {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("Total tests: %d\n", len(ts.Results))
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Total time: %v\n", totalDuration)

	if failed > 0 {
		fmt.Println("\nFailed tests:")
		for _, result := range ts.Results {
			if result.Status == "FAIL" {
				fmt.Printf("  • %s: %s\n", result.Name, result.Error)
			}
		}
	}
}

// Helper function to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Test functions

func testListBuckets(config *TestConfig) error {
	_, err := config.S3Client.ListBuckets(&s3.ListBucketsInput{})
	return err
}

func testCreateBucket(config *TestConfig) error {
	_, err := config.S3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(config.TestBucket),
	})
	if err == nil {
		config.CleanupBuckets = append(config.CleanupBuckets, config.TestBucket)
	}
	return err
}

func testHeadBucket(config *TestConfig) error {
	_, err := config.S3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(config.TestBucket),
	})
	return err
}

func testPutObject(config *TestConfig) error {
	content := "Hello, World! This is a test object."
	_, err := config.S3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(config.TestBucket),
		Key:    aws.String("test-object.txt"),
		Body:   bytes.NewReader([]byte(content)),
	})
	return err
}

func testGetObject(config *TestConfig) error {
	resp, err := config.S3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(config.TestBucket),
		Key:    aws.String("test-object.txt"),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Verify content
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	content := buf.String()

	expectedContent := "Hello, World! This is a test object."
	if content != expectedContent {
		return fmt.Errorf("content mismatch: expected %q, got %q", expectedContent, content)
	}

	return nil
}

func testHeadObject(config *TestConfig) error {
	_, err := config.S3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(config.TestBucket),
		Key:    aws.String("test-object.txt"),
	})
	return err
}

func testListObjectsV2(config *TestConfig) error {
	resp, err := config.S3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.TestBucket),
	})
	if err != nil {
		return err
	}

	if len(resp.Contents) == 0 {
		return fmt.Errorf("expected at least one object, got none")
	}

	return nil
}

func testCopyObject(config *TestConfig) error {
	copySource := fmt.Sprintf("%s/test-object.txt", config.TestBucket)
	_, err := config.S3Client.CopyObject(&s3.CopyObjectInput{
		Bucket:     aws.String(config.TestBucket),
		Key:        aws.String("copied-object.txt"),
		CopySource: aws.String(copySource),
	})
	return err
}

func testGetCopiedObject(config *TestConfig) error {
	resp, err := config.S3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(config.TestBucket),
		Key:    aws.String("copied-object.txt"),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Verify content matches original
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	content := buf.String()

	expectedContent := "Hello, World! This is a test object."
	if content != expectedContent {
		return fmt.Errorf("copied content mismatch: expected %q, got %q", expectedContent, content)
	}

	return nil
}

func testDeleteObject(config *TestConfig) error {
	_, err := config.S3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(config.TestBucket),
		Key:    aws.String("test-object.txt"),
	})
	return err
}

func testDeleteCopiedObject(config *TestConfig) error {
	_, err := config.S3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(config.TestBucket),
		Key:    aws.String("copied-object.txt"),
	})
	return err
}

func testGetDeletedObject(config *TestConfig) error {
	_, err := config.S3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(config.TestBucket),
		Key:    aws.String("test-object.txt"),
	})

	if err == nil {
		return fmt.Errorf("expected error when getting deleted object, but got none")
	}

	// Check if it's the expected "not found" error
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == s3.ErrCodeNoSuchKey || awsErr.Code() == "NoSuchKey" {
			return nil // Expected error
		}
	}

	return fmt.Errorf("unexpected error type: %v", err)
}

func testListObjectsWithPrefix(config *TestConfig) error {
	// First, create some test objects with different prefixes
	testObjects := []string{
		"prefix1/object1.txt",
		"prefix1/object2.txt",
		"prefix2/object3.txt",
		"no-prefix.txt",
	}

	for _, key := range testObjects {
		_, err := config.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(config.TestBucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader([]byte("test content")),
		})
		if err != nil {
			return fmt.Errorf("failed to create test object %s: %v", key, err)
		}
	}

	// Test listing with prefix
	resp, err := config.S3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.TestBucket),
		Prefix: aws.String("prefix1/"),
	})
	if err != nil {
		return err
	}

	if len(resp.Contents) != 2 {
		return fmt.Errorf("expected 2 objects with prefix1/, got %d", len(resp.Contents))
	}

	// Clean up test objects
	for _, key := range testObjects {
		_, _ = config.S3Client.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(config.TestBucket),
			Key:    aws.String(key),
		})
	}

	return nil
}

func testMultipleObjectOperations(config *TestConfig) error {
	objectKeys := []string{
		"multi-test/object1.txt",
		"multi-test/object2.txt",
		"multi-test/object3.txt",
	}

	// Create multiple objects
	for i, key := range objectKeys {
		content := fmt.Sprintf("Multi-object test content #%d", i+1)
		_, err := config.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(config.TestBucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader([]byte(content)),
		})
		if err != nil {
			return fmt.Errorf("failed to create object %s: %v", key, err)
		}
	}

	// List objects to verify they were created
	resp, err := config.S3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.TestBucket),
		Prefix: aws.String("multi-test/"),
	})
	if err != nil {
		return err
	}

	if len(resp.Contents) != len(objectKeys) {
		return fmt.Errorf("expected %d objects, got %d", len(objectKeys), len(resp.Contents))
	}

	// Clean up
	for _, key := range objectKeys {
		_, _ = config.S3Client.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(config.TestBucket),
			Key:    aws.String(key),
		})
	}

	return nil
}

func testDeleteBucket(config *TestConfig) error {
	return deleteBucketRecursive(config, config.TestBucket)
}

// Helper function to delete a bucket and all its contents
func deleteBucketRecursive(config *TestConfig, bucketName string) error {
	// First, list and delete all objects in the bucket
	listResp, err := config.S3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// If bucket doesn't exist, that's fine
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == s3.ErrCodeNoSuchBucket || awsErr.Code() == "NoSuchBucket" {
				return nil
			}
		}
		return err
	}

	// Delete all objects
	for _, obj := range listResp.Contents {
		_, err := config.S3Client.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    obj.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to delete object %s: %v", *obj.Key, err)
		}
	}

	// Delete the bucket
	_, err = config.S3Client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

// TestS3Compatibility runs the complete S3 API compatibility test suite
func TestS3Compatibility(t *testing.T) {
	fmt.Println("🧪 github-as-s3 S3 API Compatibility Test Suite")
	fmt.Println("================================================")
	fmt.Printf("Endpoint: %s\n", getEnv("GHS3_ENDPOINT", "http://localhost:8080"))
	fmt.Printf("Access Key: %s\n", getEnv("AWS_ACCESS_KEY_ID", "minioadmin"))
	fmt.Println()

	ts := NewTestSuite()
	defer ts.Cleanup()

	// Test execution order matters due to dependencies
	fmt.Println("🚀 Running S3 API tests...")
	fmt.Println()

	// Basic bucket operations
	ts.RunTest("ListBuckets", testListBuckets)
	ts.RunTest("CreateBucket", testCreateBucket)
	ts.RunTest("HeadBucket", testHeadBucket)

	// Object operations
	ts.RunTest("PutObject", testPutObject)
	ts.RunTest("GetObject", testGetObject)
	ts.RunTest("HeadObject", testHeadObject)
	ts.RunTest("ListObjectsV2", testListObjectsV2)

	// Copy operations
	ts.RunTest("CopyObject", testCopyObject)
	ts.RunTest("GetCopiedObject", testGetCopiedObject)

	// Delete operations
	ts.RunTest("DeleteObject", testDeleteObject)
	ts.RunTest("DeleteCopiedObject", testDeleteCopiedObject)
	ts.RunTest("GetDeletedObject", testGetDeletedObject)

	// Advanced operations
	ts.RunTest("ListObjectsWithPrefix", testListObjectsWithPrefix)
	ts.RunTest("MultipleObjectOperations", testMultipleObjectOperations)

	// Cleanup
	ts.RunTest("DeleteBucket", testDeleteBucket)

	// Print summary
	ts.PrintSummary()

	// Check if any tests failed
	failed := 0
	for _, result := range ts.Results {
		if result.Status == "FAIL" {
			failed++
		}
	}

	if failed > 0 {
		t.Errorf("%d test(s) failed", failed)
	}
}
