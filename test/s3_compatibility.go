package s3test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
	// Configuration - use the helper function from suite_test.go
	endpoint := getEnv("GHS3_ENDPOINT", "http://localhost:8080")
	accessKey := getEnv("AWS_ACCESS_KEY_ID", "minioadmin")
	secretKey := getEnv("AWS_SECRET_ACCESS_KEY", "minioadmin")
	region := getEnv("AWS_REGION", "us-east-1")

	fmt.Println("🧪 github-as-s3 S3 API Compatibility Test Suite")
	fmt.Println("================================================")
	fmt.Printf("Endpoint: %s\n", endpoint)
	fmt.Printf("Access Key: %s\n", accessKey)
	fmt.Println()

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(true),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	s3Client := s3.New(sess)
	testBucket := fmt.Sprintf("test-bucket-%d", time.Now().Unix())

	// Run tests
	fmt.Println("🚀 Running S3 API tests...")
	fmt.Println()

	passed := 0
	failed := 0

	// Test 1: List Buckets
	if runTest("ListBuckets", func() error {
		_, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
		return err
	}) {
		passed++
	} else {
		failed++
	}

	// Test 2: Create Bucket
	if runTest("CreateBucket", func() error {
		_, err := s3Client.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(testBucket),
		})
		return err
	}) {
		passed++
	} else {
		failed++
	}

	// Test 3: Head Bucket
	if runTest("HeadBucket", func() error {
		_, err := s3Client.HeadBucket(&s3.HeadBucketInput{
			Bucket: aws.String(testBucket),
		})
		return err
	}) {
		passed++
	} else {
		failed++
	}

	// Test 4: Put Object
	if runTest("PutObject", func() error {
		content := "Hello, World! This is a test object."
		_, err := s3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(testBucket),
			Key:    aws.String("test-object.txt"),
			Body:   bytes.NewReader([]byte(content)),
		})
		return err
	}) {
		passed++
	} else {
		failed++
	}

	// Test 5: Get Object
	if runTest("GetObject", func() error {
		resp, err := s3Client.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(testBucket),
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
	}) {
		passed++
	} else {
		failed++
	}

	// Test 6: List Objects V2
	if runTest("ListObjectsV2", func() error {
		resp, err := s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: aws.String(testBucket),
		})
		if err != nil {
			return err
		}

		if len(resp.Contents) == 0 {
			return fmt.Errorf("expected at least one object, got none")
		}
		return nil
	}) {
		passed++
	} else {
		failed++
	}

	// Test 7: Copy Object
	if runTest("CopyObject", func() error {
		copySource := fmt.Sprintf("%s/test-object.txt", testBucket)
		_, err := s3Client.CopyObject(&s3.CopyObjectInput{
			Bucket:     aws.String(testBucket),
			Key:        aws.String("copied-object.txt"),
			CopySource: aws.String(copySource),
		})
		return err
	}) {
		passed++
	} else {
		failed++
	}

	// Test 8: Delete Object
	if runTest("DeleteObject", func() error {
		_, err := s3Client.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(testBucket),
			Key:    aws.String("test-object.txt"),
		})
		return err
	}) {
		passed++
	} else {
		failed++
	}

	// Test 9: Delete Copied Object
	if runTest("DeleteCopiedObject", func() error {
		_, err := s3Client.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(testBucket),
			Key:    aws.String("copied-object.txt"),
		})
		return err
	}) {
		passed++
	} else {
		failed++
	}

	// Test 10: Delete Bucket
	if runTest("DeleteBucket", func() error {
		_, err := s3Client.DeleteBucket(&s3.DeleteBucketInput{
			Bucket: aws.String(testBucket),
		})
		return err
	}) {
		passed++
	} else {
		failed++
	}

	// Print summary
	fmt.Println()
	fmt.Println("📊 Test Summary")
	fmt.Println("===============")
	fmt.Printf("Total tests: %d\n", passed+failed)
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func runTest(name string, testFunc func() error) bool {
	start := time.Now()
	fmt.Printf("  • %-20s ", name)

	err := testFunc()
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ FAIL (%v) - %s\n", duration, err.Error())
		return false
	} else {
		fmt.Printf("✅ PASS (%v)\n", duration)
		return true
	}
}
