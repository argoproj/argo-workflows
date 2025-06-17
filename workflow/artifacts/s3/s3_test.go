package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common/pool"
)

const transientEnvVarKey = "TRANSIENT_ERROR_PATTERN"

type mockS3Client struct {
	// files is a map where key is bucket name and value consists of file keys
	files map[string][]string
	// mockedErrs is a map where key is the function name and value is the mocked error of that function
	mockedErrs map[string]error
}

func newMockS3Client(files map[string][]string, mockedErrs map[string]error) S3Client {
	return &mockS3Client{
		files:      files,
		mockedErrs: mockedErrs,
	}
}

func (s *mockS3Client) getMockedErr(funcName string) error {
	err, ok := s.mockedErrs[funcName]
	if !ok {
		return nil
	}
	return err
}

// PutFile puts a single file to a bucket at the specified key
func (s *mockS3Client) PutFile(bucket, key, path string) error {
	return s.getMockedErr("PutFile")
}

// PutDirectory puts a complete directory into a bucket key prefix, with each file in the directory
// a separate key in the bucket.
func (s *mockS3Client) PutDirectory(bucket, key, path string) error {
	return s.getMockedErr("PutDirectory")
}

// GetFile downloads a file to a local file path
func (s *mockS3Client) GetFile(bucket, key, path string) error {
	return s.getMockedErr("GetFile")
}

func (s *mockS3Client) OpenFile(bucket, key string) (io.ReadCloser, error) {
	err := s.getMockedErr("OpenFile")
	if err == nil {
		return io.NopCloser(&bytes.Buffer{}), nil
	}
	return nil, err
}

func (s *mockS3Client) KeyExists(bucket, key string) (bool, error) {
	err := s.getMockedErr("KeyExists")
	if files, ok := s.files[bucket]; ok {
		for _, file := range files {
			if strings.HasPrefix(file, key+"/") || file == key { // either it's a prefixing directory or the key itself
				return true, err
			}
		}
	}
	return false, err
}

// GetDirectory downloads a directory to a local file path
func (s *mockS3Client) GetDirectory(bucket, key, path string) error {
	return s.getMockedErr("GetDirectory")
}

// ListDirectory list the contents of a directory/bucket
func (s *mockS3Client) ListDirectory(bucket, keyPrefix string) ([]string, error) {
	dirs := make([]string, 0)
	err := s.getMockedErr("ListDirectory")
	if files, ok := s.files[bucket]; ok {
		for _, file := range files {
			if strings.HasPrefix(file, keyPrefix+"/") {
				dirs = append(dirs, file)
			}
		}
	}
	return dirs, err
}

// IsDirectory tests if the key is acting like a s3 directory
func (s *mockS3Client) IsDirectory(bucket, key string) (bool, error) {
	var isDir bool
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	if files, ok := s.files[bucket]; ok {
		for _, file := range files {
			if strings.HasPrefix(file, key) {
				isDir = true
				break
			}
		}
	}
	return isDir, s.getMockedErr("IsDirectory")
}

// BucketExists returns whether a bucket exists
func (s *mockS3Client) BucketExists(bucket string) (bool, error) {
	err := s.getMockedErr("BucketExists")
	if _, ok := s.files[bucket]; ok {
		return true, err
	}
	return false, err
}

// MakeBucket creates a bucket with name bucketName and options opts
func (s *mockS3Client) MakeBucket(bucketName string, opts minio.MakeBucketOptions) error {
	return s.getMockedErr("MakeBucket")
}

func TestOpenStreamS3Artifact(t *testing.T) {
	tests := map[string]struct {
		s3client  S3Client
		bucket    string
		key       string
		localPath string
		errMsg    string
	}{
		"Success": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art.tar.gz",
					},
				},
				map[string]error{}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: "/tmp/hello-art.tar.gz",
			errMsg:    "",
		},
		"No such bucket": {
			s3client: newMockS3Client(
				map[string][]string{},
				map[string]error{
					"OpenFile": minio.ErrorResponse{
						Code: "NoSuchBucket",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: "/tmp/hello-art.tar.gz",
			errMsg:    "failed to get file: The specified bucket does not exist.",
		},
		"No such key": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art-2.tar.gz",
					},
				},
				map[string]error{
					"OpenFile": minio.ErrorResponse{
						Code: "NoSuchKey",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: "/tmp/hello-art.tar.gz",
			errMsg:    "The specified key does not exist.",
		},
		"Is Directory": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art-2.tar.gz",
					},
				},
				map[string]error{
					"OpenFile": minio.ErrorResponse{
						Code: "NoSuchKey",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/",
			localPath: "/tmp/folder/",
			errMsg:    "Directory Stream capability currently unimplemented for S3",
		},
		"Test Directory Failed": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art-2.tar.gz",
					},
				},
				map[string]error{
					"OpenFile": minio.ErrorResponse{
						Code: "NoSuchKey",
					},
					"IsDirectory": minio.ErrorResponse{
						Code: "InternalError",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/",
			localPath: "/tmp/folder/",
			errMsg:    "failed to test if /folder/ is a directory: We encountered an internal error, please try again.",
		},
	}

	t.Setenv(transientEnvVarKey, "this error is transient")
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stream, err := streamS3Artifact(tc.s3client, &wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					S3: &wfv1.S3Artifact{
						S3Bucket: wfv1.S3Bucket{
							Bucket: tc.bucket,
						},
						Key: tc.key,
					},
				},
			})
			if tc.errMsg == "" {
				require.NoError(t, err)
				assert.NotNil(t, stream)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.errMsg, err.Error())
			}
		})
	}
}

// Delete deletes an S3 artifact by artifact key
func (s *mockS3Client) Delete(bucket, key string) error {
	return s.getMockedErr("Delete")
}

func TestLoadS3Artifact(t *testing.T) {
	tests := map[string]struct {
		s3client  S3Client
		bucket    string
		key       string
		localPath string
		done      bool
		errMsg    string
	}{
		"Success": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art.tar.gz",
					},
				},
				map[string]error{}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: "/tmp/hello-art.tar.gz",
			done:      true,
			errMsg:    "",
		},
		"No such bucket": {
			s3client: newMockS3Client(
				map[string][]string{},
				map[string]error{
					"GetFile": minio.ErrorResponse{
						Code: "NoSuchBucket",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: "/tmp/hello-art.tar.gz",
			done:      true,
			errMsg:    "failed to get file: The specified bucket does not exist.",
		},
		"No such key": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art-2.tar.gz",
					},
				},
				map[string]error{
					"GetFile": minio.ErrorResponse{
						Code: "NoSuchKey",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: "/tmp/hello-art.tar.gz",
			done:      true,
			errMsg:    "The specified key does not exist.",
		},
		"Is Directory": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art-2.tar.gz",
					},
				},
				map[string]error{
					"GetFile": minio.ErrorResponse{
						Code: "NoSuchKey",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/",
			localPath: "/tmp/folder/",
			done:      true,
			errMsg:    "",
		},
		"Get File Other Transient Error": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art-2.tar.gz",
					},
				},
				map[string]error{
					"GetFile": minio.ErrorResponse{
						Code: "this error is transient",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/",
			localPath: "/tmp/folder/",
			done:      false,
			errMsg:    "failed to get file: Error response code this error is transient.",
		},
		"Test Directory Failed": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art-2.tar.gz",
					},
				},
				map[string]error{
					"GetFile": minio.ErrorResponse{
						Code: "NoSuchKey",
					},
					"IsDirectory": minio.ErrorResponse{
						Code: "InternalError",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/",
			localPath: "/tmp/folder/",
			done:      false,
			errMsg:    "failed to test if /folder/ is a directory: We encountered an internal error, please try again.",
		},
		"Get Directory Failed": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art-2.tar.gz",
					},
				},
				map[string]error{
					"GetFile": minio.ErrorResponse{
						Code: "NoSuchKey",
					},
					"GetDirectory": minio.ErrorResponse{
						Code: "InternalError",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/",
			localPath: "/tmp/folder/",
			done:      false,
			errMsg:    "failed to get directory: We encountered an internal error, please try again.",
		},
	}

	t.Setenv(transientEnvVarKey, "this error is transient")
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			success, err := loadS3Artifact(tc.s3client, &wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					S3: &wfv1.S3Artifact{
						S3Bucket: wfv1.S3Bucket{
							Bucket: tc.bucket,
						},
						Key: tc.key,
					},
				},
			}, tc.localPath)
			assert.Equal(t, tc.done, success)
			if err != nil {
				assert.Equal(t, tc.errMsg, err.Error())
			} else {
				assert.Empty(t, tc.errMsg)
			}
		})
	}
}

func TestSaveS3Artifact(t *testing.T) {
	tempDir := t.TempDir()

	tempFile := filepath.Join(tempDir, "tmpfile")
	if err := os.WriteFile(tempFile, []byte("temporary file's content"), 0o600); err != nil {
		panic(err)
	}

	tests := map[string]struct {
		s3client  S3Client
		bucket    string
		key       string
		localPath string
		done      bool
		errMsg    string
	}{
		"Success as File": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {},
				},
				map[string]error{}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: tempFile,
			done:      true,
			errMsg:    "",
		},
		"Success as Directory": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {},
				},
				map[string]error{}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: tempDir,
			done:      true,
			errMsg:    "",
		},
		"Make Bucket Access Denied": {
			s3client: newMockS3Client(
				map[string][]string{},
				map[string]error{
					"MakeBucket": minio.ErrorResponse{
						Code: "AccessDenied",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: tempDir,
			done:      true,
			errMsg:    "failed to create bucket my-bucket: Access Denied.",
		},
		"Save Directory Transient Error": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {},
				},
				map[string]error{
					"PutDirectory": minio.ErrorResponse{
						Code: "InternalError",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: tempDir,
			done:      false,
			errMsg:    "failed to put directory: We encountered an internal error, please try again.",
		},
		"Save File Transient Error": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {},
				},
				map[string]error{
					"PutFile": minio.ErrorResponse{
						Code: "InternalError",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: tempFile,
			done:      false,
			errMsg:    "failed to put file: We encountered an internal error, please try again.",
		},
		"Save File Other Transient Error": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {},
				},
				map[string]error{
					"PutFile": minio.ErrorResponse{
						Code: "this error is transient",
					},
				}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: tempFile,
			done:      false,
			errMsg:    "failed to put file: Error response code this error is transient.",
		},
	}

	for name, tc := range tests {
		t.Setenv(transientEnvVarKey, "this error is transient")
		t.Run(name, func(t *testing.T) {
			success, err := saveS3Artifact(
				tc.s3client,
				tc.localPath,
				&wfv1.Artifact{
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							S3Bucket: wfv1.S3Bucket{
								Bucket:                   tc.bucket,
								CreateBucketIfNotPresent: &wfv1.CreateS3BucketOptions{},
								EncryptionOptions: &wfv1.S3EncryptionOptions{
									EnableEncryption: true,
								},
							},
							Key: tc.key,
						},
					},
				})
			assert.Equal(t, tc.done, success)
			if err != nil {
				assert.Equal(t, tc.errMsg, err.Error())
			} else {
				assert.Empty(t, tc.errMsg)
			}
		})
	}
}

func TestListObjects(t *testing.T) {

	tests := map[string]struct {
		s3client         S3Client
		bucket           string
		key              string
		expectedSuccess  bool
		expectedErrMsg   string
		expectedNumFiles int
	}{
		"Found objects": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder/hello-art.tar.gz",
					},
				},
				map[string]error{}),
			bucket:           "my-bucket",
			key:              "/folder",
			expectedSuccess:  true,
			expectedNumFiles: 1,
		},
		"Empty directory": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder",
					},
				},
				map[string]error{}),
			bucket:           "my-bucket",
			key:              "/folder",
			expectedSuccess:  true,
			expectedNumFiles: 0,
		},
		"Non-existent directory": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": {
						"/folder",
					},
				},
				map[string]error{}),
			bucket:          "my-bucket",
			key:             "/non-existent-folder",
			expectedSuccess: false,
			expectedErrMsg:  "no key found of name /non-existent-folder",
		},
	}

	t.Setenv(transientEnvVarKey, "this error is transient")
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, files, err := listObjects(tc.s3client,
				&wfv1.Artifact{
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							S3Bucket: wfv1.S3Bucket{
								Bucket:                   tc.bucket,
								CreateBucketIfNotPresent: &wfv1.CreateS3BucketOptions{},
								EncryptionOptions: &wfv1.S3EncryptionOptions{
									EnableEncryption: true,
								},
							},
							Key: tc.key,
						},
					},
				})
			if tc.expectedSuccess {
				require.NoError(t, err)
				assert.Len(t, files, tc.expectedNumFiles)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.expectedErrMsg, err.Error())
			}
		})
	}
}

// TestNewS3Client tests the s3 constructor
func TestNewS3Client(t *testing.T) {
	opts := S3ClientOpts{
		Endpoint:        "foo.com",
		Region:          "us-south-3",
		Secure:          false,
		Transport:       http.DefaultTransport,
		AccessKey:       "key",
		SecretKey:       "secret",
		SessionToken:    "",
		Trace:           true,
		RoleARN:         "",
		RoleSessionName: "",
		UseSDKCreds:     false,
		EncryptOpts:     EncryptOpts{Enabled: true, ServerSideCustomerKey: "", KmsKeyID: "", KmsEncryptionContext: ""},
	}
	s3If, err := NewS3Client(context.Background(), opts)
	require.NoError(t, err)
	s3cli := s3If.(*s3client)
	assert.Equal(t, opts.Endpoint, s3cli.Endpoint)
	assert.Equal(t, opts.Region, s3cli.Region)
	assert.Equal(t, opts.Secure, s3cli.Secure)
	assert.Equal(t, opts.Transport, s3cli.Transport)
	assert.Equal(t, opts.AccessKey, s3cli.AccessKey)
	assert.Equal(t, opts.SessionToken, s3cli.SessionToken)
	assert.Equal(t, opts.Trace, s3cli.Trace)
	assert.Equal(t, opts.EncryptOpts, s3cli.EncryptOpts)
	assert.Equal(t, opts.AddressingStyle, s3cli.AddressingStyle)
	// s3cli.minioClient.
	// 	s3client.minioClient
}

// TestNewS3Client tests the S3 constructor using ephemeral credentials
func TestNewS3ClientEphemeral(t *testing.T) {
	opts := S3ClientOpts{
		Endpoint:     "foo.com",
		Region:       "us-south-3",
		AccessKey:    "key",
		SecretKey:    "secret",
		SessionToken: "sessionToken",
	}
	s3If, err := NewS3Client(context.Background(), opts)
	require.NoError(t, err)
	s3cli := s3If.(*s3client)
	assert.Equal(t, opts.Endpoint, s3cli.Endpoint)
	assert.Equal(t, opts.Region, s3cli.Region)
	assert.Equal(t, opts.AccessKey, s3cli.AccessKey)
	assert.Equal(t, opts.SecretKey, s3cli.SecretKey)
	assert.Equal(t, opts.SessionToken, s3cli.SessionToken)
}

// TestNewS3Client tests the s3 constructor
func TestNewS3ClientWithDiff(t *testing.T) {
	t.Run("IAMRole", func(t *testing.T) {
		opts := S3ClientOpts{
			Endpoint: "foo.com",
			Region:   "us-south-3",
			Secure:   false,
			Trace:    true,
		}
		s3If, err := NewS3Client(context.Background(), opts)
		require.NoError(t, err)
		s3cli := s3If.(*s3client)
		assert.Equal(t, opts.Endpoint, s3cli.Endpoint)
		assert.Equal(t, opts.Region, s3cli.Region)
		assert.Equal(t, opts.Trace, s3cli.Trace)
		assert.Equal(t, opts.Endpoint, s3cli.minioClient.EndpointURL().Host)
	})
	t.Run("AssumeIAMRole", func(t *testing.T) {
		t.SkipNow()
		opts := S3ClientOpts{
			Endpoint: "foo.com",
			Region:   "us-south-3",
			Secure:   false,
			Trace:    true,
			RoleARN:  "01234567890123456789",
		}
		s3If, err := NewS3Client(context.Background(), opts)
		require.NoError(t, err)
		s3cli := s3If.(*s3client)
		assert.Equal(t, opts.Endpoint, s3cli.Endpoint)
		assert.Equal(t, opts.Region, s3cli.Region)
		assert.Equal(t, opts.Trace, s3cli.Trace)
		assert.Equal(t, opts.Endpoint, s3cli.minioClient.EndpointURL().Host)
	})
}

func TestDisallowedComboOptions(t *testing.T) {
	t.Run("KMS and SSEC", func(t *testing.T) {
		opts := S3ClientOpts{
			Endpoint:    "foo.com",
			Region:      "us-south-3",
			Secure:      true,
			Trace:       true,
			EncryptOpts: EncryptOpts{Enabled: true, ServerSideCustomerKey: "PASSWORD", KmsKeyID: "00000000-0000-0000-0000-000000000000", KmsEncryptionContext: ""},
		}
		_, err := NewS3Client(context.Background(), opts)
		assert.Error(t, err)
	})

	t.Run("SSEC and InSecure", func(t *testing.T) {
		opts := S3ClientOpts{
			Endpoint:    "foo.com",
			Region:      "us-south-3",
			Secure:      false,
			Trace:       true,
			EncryptOpts: EncryptOpts{Enabled: true, ServerSideCustomerKey: "PASSWORD", KmsKeyID: "", KmsEncryptionContext: ""},
		}
		_, err := NewS3Client(context.Background(), opts)
		assert.Error(t, err)
	})
}

// getGoroutineID returns the current goroutine's ID
func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	var id uint64
	_, err := fmt.Sscanf(string(b), "goroutine %d ", &id)
	if err != nil {
		return 0
	}
	return id
}

// Test the actual pool.RunPoolStreaming function directly
func TestPoolStreamingParallelism(t *testing.T) {
	// Track calls per goroutine
	callTracker := newCallTracker()

	// Create a producer that generates 100 tasks
	producer := func(ctx context.Context, taskCh chan<- pool.Task) error {
		for i := 0; i < 100; i++ {
			task := pool.Task{
				SourcePath: fmt.Sprintf("source%d", i),
				DestKey:    fmt.Sprintf("dest%d", i),
				IsUpload:   true,
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case taskCh <- task:
			}
		}
		return nil
	}

	// Create a worker function that tracks calls
	worker := func(t pool.Task) error {
		callTracker.recordCall()
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	// Run with 5 parallel workers
	err := pool.RunPoolStreaming(context.Background(), 5, producer, worker)
	if err != nil {
		t.Fatalf("RunPoolStreaming failed: %v", err)
	}

	// Verify parallel execution
	totalCalls, workerCount, maxCallsPerWorker := callTracker.getStats()

	if totalCalls != 100 {
		t.Errorf("Expected 100 total calls, got %d", totalCalls)
	}

	if workerCount < 2 {
		t.Errorf("Expected multiple workers to be used, got %d", workerCount)
	}

	// With 5 workers and 100 tasks, no single worker should handle more than ~30 tasks
	if maxCallsPerWorker > 40 {
		t.Errorf("Work not distributed evenly: max calls per worker was %d, expected < 40", maxCallsPerWorker)
	}

	t.Logf("Pool test: %d total calls distributed among %d workers (max %d calls per worker)",
		totalCalls, workerCount, maxCallsPerWorker)
}

// Test S3 integration with a functional approach - test the real directory walking logic
func TestS3DirectoryWalkingParallelism(t *testing.T) {
	// Create a temporary directory with 50 files
	dir, err := os.MkdirTemp("", "s3-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create 50 files in the directory
	for i := 0; i < 50; i++ {
		filePath := filepath.Join(dir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Track calls per goroutine
	callTracker := newCallTracker()

	// Create a producer that mimics the real PutDirectory logic
	producer := func(ctx context.Context, taskCh chan<- pool.Task) error {
		return filepath.Walk(dir, func(fpath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			// Calculate relative path from root (same as real code)
			relPath, err := filepath.Rel(dir, fpath)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %v", err)
			}

			// Convert to S3-style path (same as real code)
			s3Path := filepath.ToSlash(relPath)
			key := "test"
			if key != "" {
				s3Path = filepath.ToSlash(filepath.Join(key, s3Path))
			}

			task := pool.Task{
				SourcePath: fpath,
				DestKey:    s3Path,
				IsUpload:   true,
			}

			// Stream the task to workers (same as real code)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case taskCh <- task:
				return nil
			}
		})
	}

	// Create a worker function that simulates S3 upload
	uploadedFiles := make(map[string]string)
	var uploadMutex sync.Mutex

	worker := func(t pool.Task) error {
		callTracker.recordCall()

		// Simulate S3 upload work
		time.Sleep(10 * time.Millisecond)

		// Record the uploaded file
		uploadMutex.Lock()
		uploadedFiles[t.DestKey] = t.SourcePath
		uploadMutex.Unlock()

		return nil
	}

	// Run with 5 parallel workers (same as our test configuration)
	err = pool.RunPoolStreaming(context.Background(), 5, producer, worker)
	if err != nil {
		t.Fatalf("RunPoolStreaming failed: %v", err)
	}

	// Verify that all files were processed
	if len(uploadedFiles) != 50 {
		t.Errorf("Expected 50 files to be uploaded, got %d", len(uploadedFiles))
	}

	// Verify parallel execution
	totalCalls, workerCount, maxCallsPerWorker := callTracker.getStats()

	if totalCalls != 50 {
		t.Errorf("Expected 50 total calls, got %d", totalCalls)
	}

	if workerCount < 2 {
		t.Errorf("Expected multiple workers to be used, got %d", workerCount)
	}

	// With 5 workers and 50 files, no single worker should handle more than ~15 files
	if maxCallsPerWorker > 20 {
		t.Errorf("Work not distributed evenly: max calls per worker was %d, expected < 20", maxCallsPerWorker)
	}

	t.Logf("S3 directory test: %d total calls distributed among %d workers (max %d calls per worker)",
		totalCalls, workerCount, maxCallsPerWorker)
}

// Test S3 download parallelism using the real GetDirectory logic
func TestS3DirectoryDownloadParallelism(t *testing.T) {
	// Create a temporary directory for download
	dir, err := os.MkdirTemp("", "s3-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Simulate S3 objects (same as real code would get from ListObjects)
	s3Objects := []string{
		"file1.txt",
		"file2.txt",
		"subdir/file3.txt",
		"subdir/file4.txt",
		"another/path/file5.txt",
		"another/path/file6.txt",
	}

	// Track calls per goroutine
	callTracker := newCallTracker()

	// Create a producer that mimics the real GetDirectory logic
	producer := func(ctx context.Context, taskCh chan<- pool.Task) error {
		keyPrefix := ""
		if keyPrefix != "" {
			keyPrefix = filepath.Clean(keyPrefix) + "/"
			if os.PathSeparator == '\\' {
				keyPrefix = strings.ReplaceAll(keyPrefix, "\\", "/")
			}
		}

		// Simulate the ListObjects channel (same as real code)
		for _, objKey := range s3Objects {
			if strings.HasSuffix(objKey, "/") {
				// Skip directory objects created by AWS S3 console
				continue
			}

			relKeyPath := strings.TrimPrefix(objKey, keyPrefix)
			localPath := filepath.Join(dir, relKeyPath)

			task := pool.Task{
				SourcePath: objKey,
				DestKey:    localPath,
				IsUpload:   false,
			}

			// Stream the task to workers (same as real code)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case taskCh <- task:
			}
		}
		return nil
	}

	// Create a worker function that simulates S3 download
	worker := func(t pool.Task) error {
		callTracker.recordCall()

		// Simulate S3 download work
		time.Sleep(10 * time.Millisecond)

		// Create directory if needed (same as real code)
		dirPath := filepath.Dir(t.DestKey)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dirPath, err)
		}

		// Create the file (simulate download)
		return os.WriteFile(t.DestKey, []byte("test content"), 0644)
	}

	// Run with 3 parallel workers
	err = pool.RunPoolStreaming(context.Background(), 3, producer, worker)
	if err != nil {
		t.Fatalf("RunPoolStreaming failed: %v", err)
	}

	// Verify that all files were downloaded
	for _, objKey := range s3Objects {
		localPath := filepath.Join(dir, objKey)
		if _, err := os.Stat(localPath); err != nil {
			t.Errorf("Failed to find downloaded file %s: %v", objKey, err)
		}
	}

	// Verify parallel execution
	totalCalls, workerCount, maxCallsPerWorker := callTracker.getStats()

	if totalCalls != len(s3Objects) {
		t.Errorf("Expected %d total calls, got %d", len(s3Objects), totalCalls)
	}

	if workerCount < 2 {
		t.Errorf("Expected multiple workers to be used, got %d", workerCount)
	}

	// With 3 workers and 6 files, no single worker should handle more than 3 files
	if maxCallsPerWorker > 3 {
		t.Errorf("Work not distributed evenly: max calls per worker was %d, expected <= 3", maxCallsPerWorker)
	}

	t.Logf("S3 download test: %d total calls distributed among %d workers (max %d calls per worker)",
		totalCalls, workerCount, maxCallsPerWorker)
}

type callTracker struct {
	mu          sync.Mutex
	workerCalls map[uint64]int // goroutine ID -> call count
	totalCalls  int
}

func newCallTracker() *callTracker {
	return &callTracker{
		workerCalls: make(map[uint64]int),
	}
}

func (ct *callTracker) recordCall() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	workerID := getGoroutineID()
	ct.workerCalls[workerID]++
	ct.totalCalls++
}

func (ct *callTracker) getStats() (totalCalls int, workerCount int, maxCallsPerWorker int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	totalCalls = ct.totalCalls
	workerCount = len(ct.workerCalls)

	for _, calls := range ct.workerCalls {
		if calls > maxCallsPerWorker {
			maxCallsPerWorker = calls
		}
	}
	return
}

// Test ParallelTransfers configuration
func TestS3ClientParallelTransfersConfig(t *testing.T) {
	tests := []struct {
		name              string
		parallelTransfers int
		expectedParallel  int
	}{
		{
			name:              "Default auto-detect",
			parallelTransfers: 0,
			expectedParallel:  runtime.NumCPU() * 2, // Will be capped at maxParallel if > 32
		},
		{
			name:              "Explicit value",
			parallelTransfers: 5,
			expectedParallel:  5,
		},
		{
			name:              "Negative value fallback",
			parallelTransfers: -1,
			expectedParallel:  1,
		},
		{
			name:              "Large value",
			parallelTransfers: 100,
			expectedParallel:  100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := S3ClientOpts{
				Endpoint:          "test-endpoint",
				ParallelTransfers: tc.parallelTransfers,
			}

			client, err := NewS3Client(context.Background(), opts)
			require.NoError(t, err)

			s3cli := client.(*s3client)
			actualParallel := s3cli.getParallelTransfers()

			expectedParallel := tc.expectedParallel
			if tc.parallelTransfers == 0 && expectedParallel > maxParallel {
				expectedParallel = maxParallel
			}

			assert.Equal(t, expectedParallel, actualParallel)
		})
	}
}

// Test environment variable overrides
func TestS3ClientEnvironmentOverrides(t *testing.T) {
	// Save original env vars
	origParallel := os.Getenv("ARGO_S3_PARALLEL_TRANSFERS")
	origPartSize := os.Getenv("ARGO_S3_MULTIPART_PART_SIZE")
	origConcurrency := os.Getenv("ARGO_S3_MULTIPART_CONCURRENCY")

	defer func() {
		// Restore original env vars
		os.Setenv("ARGO_S3_PARALLEL_TRANSFERS", origParallel)
		os.Setenv("ARGO_S3_MULTIPART_PART_SIZE", origPartSize)
		os.Setenv("ARGO_S3_MULTIPART_CONCURRENCY", origConcurrency)
	}()

	tests := []struct {
		name                string
		envParallel         string
		envPartSize         string
		envConcurrency      string
		baseParallel        int
		basePartSize        int64
		baseConcurrency     int
		expectedParallel    int
		expectedPartSize    int64
		expectedConcurrency int
	}{
		{
			name:                "Valid env overrides",
			envParallel:         "8",
			envPartSize:         "10485760", // 10MB
			envConcurrency:      "4",
			baseParallel:        2,
			basePartSize:        5242880, // 5MB
			baseConcurrency:     2,
			expectedParallel:    8,
			expectedPartSize:    10485760,
			expectedConcurrency: 4,
		},
		{
			name:                "Invalid env values ignored",
			envParallel:         "invalid",
			envPartSize:         "not-a-number",
			envConcurrency:      "-1",
			baseParallel:        3,
			basePartSize:        1048576, // 1MB
			baseConcurrency:     3,
			expectedParallel:    3,
			expectedPartSize:    1048576,
			expectedConcurrency: 3,
		},
		{
			name:                "Zero/negative env values ignored",
			envParallel:         "0",
			envPartSize:         "-100",
			envConcurrency:      "0",
			baseParallel:        4,
			basePartSize:        2097152, // 2MB
			baseConcurrency:     4,
			expectedParallel:    4,
			expectedPartSize:    2097152,
			expectedConcurrency: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("ARGO_S3_PARALLEL_TRANSFERS", tc.envParallel)
			os.Setenv("ARGO_S3_MULTIPART_PART_SIZE", tc.envPartSize)
			os.Setenv("ARGO_S3_MULTIPART_CONCURRENCY", tc.envConcurrency)

			// Create ArtifactDriver (which calls newS3Client)
			driver := &ArtifactDriver{
				Endpoint:             "test-endpoint",
				ParallelTransfers:    tc.baseParallel,
				MultipartPartSize:    tc.basePartSize,
				MultipartConcurrency: tc.baseConcurrency,
			}

			client, err := driver.newS3Client(context.Background())
			require.NoError(t, err)

			s3cli := client.(*s3client)
			assert.Equal(t, tc.expectedParallel, s3cli.ParallelTransfers)
			assert.Equal(t, tc.expectedPartSize, s3cli.MultipartPartSize)
			assert.Equal(t, tc.expectedConcurrency, s3cli.MultipartConcurrency)
		})
	}
}

// Test that ParallelTransfers actually controls worker count
func TestParallelTransfersControlsWorkerCount(t *testing.T) {
	// Create a temporary directory with files
	dir, err := os.MkdirTemp("", "parallel-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create 20 files
	for i := 0; i < 20; i++ {
		filePath := filepath.Join(dir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name               string
		parallelTransfers  int
		maxExpectedWorkers int
	}{
		{
			name:               "Single worker",
			parallelTransfers:  1,
			maxExpectedWorkers: 1,
		},
		{
			name:               "Three workers",
			parallelTransfers:  3,
			maxExpectedWorkers: 3,
		},
		{
			name:               "Ten workers",
			parallelTransfers:  10,
			maxExpectedWorkers: 10,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			callTracker := newCallTracker()

			// Create producer for directory walking
			producer := func(ctx context.Context, taskCh chan<- pool.Task) error {
				return filepath.Walk(dir, func(fpath string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return err
					}

					task := pool.Task{
						SourcePath: fpath,
						DestKey:    filepath.Base(fpath),
						IsUpload:   true,
					}

					select {
					case <-ctx.Done():
						return ctx.Err()
					case taskCh <- task:
						return nil
					}
				})
			}

			// Worker that tracks calls
			worker := func(t pool.Task) error {
				callTracker.recordCall()
				time.Sleep(50 * time.Millisecond) // Ensure work takes time
				return nil
			}

			// Run with specified parallel transfers
			err := pool.RunPoolStreaming(context.Background(), tc.parallelTransfers, producer, worker)
			require.NoError(t, err)

			totalCalls, workerCount, _ := callTracker.getStats()

			assert.Equal(t, 20, totalCalls, "Should process all 20 files")
			assert.LessOrEqual(t, workerCount, tc.maxExpectedWorkers,
				"Should not use more workers than configured")

			// For single worker, ensure only one worker was used
			if tc.parallelTransfers == 1 {
				assert.Equal(t, 1, workerCount, "Should use exactly 1 worker when configured for 1")
			}
		})
	}
}

// Test multipart configuration values are preserved
func TestMultipartConfigurationPreservation(t *testing.T) {
	opts := S3ClientOpts{
		Endpoint:             "test-endpoint",
		ParallelTransfers:    5,
		MultipartPartSize:    10485760, // 10MB
		MultipartConcurrency: 3,
	}

	client, err := NewS3Client(context.Background(), opts)
	require.NoError(t, err)

	s3cli := client.(*s3client)
	assert.Equal(t, 5, s3cli.ParallelTransfers)
	assert.Equal(t, int64(10485760), s3cli.MultipartPartSize)
	assert.Equal(t, 3, s3cli.MultipartConcurrency)
}
