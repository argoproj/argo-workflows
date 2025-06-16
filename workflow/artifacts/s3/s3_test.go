package s3

import (
	"bytes"
	"context"
	crand "crypto/rand"
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
	mockedErrs map[string]interface{}
	// workerCalls tracks the number of calls per worker (goroutine ID)
	workerCalls map[uint64]int
	// workerCallsMutex protects workerCalls
	workerCallsMutex sync.Mutex
	// workerCount tracks the number of unique workers used
	workerCount int
	// workerCountMutex protects workerCount
	workerCountMutex sync.Mutex
}

func newMockS3Client(files map[string][]string, mockedErrs map[string]interface{}) S3Client {
	return &mockS3Client{
		files:       files,
		mockedErrs:  mockedErrs,
		workerCalls: make(map[uint64]int),
	}
}

func (s *mockS3Client) getMockedErr(funcName string) error {
	err, ok := s.mockedErrs[funcName]
	if !ok {
		return nil
	}
	if fn, ok := err.(func() error); ok {
		return fn()
	}
	if err, ok := err.(error); ok {
		return err
	}
	return nil
}

// PutFile puts a single file to a bucket at the specified key
func (s *mockS3Client) PutFile(bucket, key, path string) error {
	return s.getMockedErr("PutFile")
}

// PutDirectory puts a complete directory into a bucket key prefix, with each file in the directory
// a separate key in the bucket.
func (s *mockS3Client) PutDirectory(bucket, key, path string) error {
	// If a mock error is set for PutDirectory, return it immediately
	if err := s.getMockedErr("PutDirectory"); err != nil {
		return err
	}
	// Ensure the path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat directory %s: %v", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %s is not a directory", path)
	}

	// Collect all files to upload
	var tasks []pool.Task
	err = filepath.Walk(path, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Calculate relative path from root
		relPath, err := filepath.Rel(path, fpath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}

		// Convert to S3-style path
		s3Path := filepath.ToSlash(relPath)
		if key != "" {
			s3Path = filepath.ToSlash(filepath.Join(key, s3Path))
		}

		tasks = append(tasks, pool.Task{
			SourcePath: fpath,
			DestKey:    s3Path,
			IsUpload:   true,
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory: %v", err)
	}

	// Run parallel uploads using the worker pool
	return pool.RunPool(context.Background(), 4, tasks, func(t pool.Task) error {
		return s.PutFile(bucket, t.DestKey, t.SourcePath)
	})
}

// GetFile downloads a file to a local file path
func (s *mockS3Client) GetFile(bucket, key, path string) error {
	if err := s.getMockedErr("GetFile"); err != nil {
		return err
	}

	// Get current goroutine ID to identify worker
	workerID := getGoroutineID()
	s.workerCountMutex.Lock()
	if _, exists := s.workerCalls[workerID]; !exists {
		s.workerCount++
	}
	s.workerCountMutex.Unlock()

	s.workerCallsMutex.Lock()
	s.workerCalls[workerID]++
	s.workerCallsMutex.Unlock()

	// Simulate some work to ensure parallel execution
	time.Sleep(time.Millisecond)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %v", path, err)
	}

	// Create a test file with some content
	if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
		return fmt.Errorf("failed to create test file %s: %v", path, err)
	}

	return nil
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
	// If a mock error is set for GetDirectory, return it immediately
	if err := s.getMockedErr("GetDirectory"); err != nil {
		return err
	}

	// Get list of files to download
	keys, err := s.ListDirectory(bucket, key)
	if err != nil {
		return err
	}

	// Create tasks for parallel download
	var tasks []pool.Task
	for _, objKey := range keys {
		relKeyPath := strings.TrimPrefix(objKey, key)
		localPath := filepath.Join(path, relKeyPath)

		// Create directory if needed
		dirPath := filepath.Dir(localPath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dirPath, err)
		}

		tasks = append(tasks, pool.Task{
			SourcePath: objKey,
			DestKey:    localPath,
			IsUpload:   false,
		})
	}

	// Run parallel downloads using the worker pool
	return pool.RunPool(context.Background(), 4, tasks, func(t pool.Task) error {
		return s.GetFile(bucket, t.SourcePath, t.DestKey)
	})
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
				map[string]interface{}{}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: "/tmp/hello-art.tar.gz",
			errMsg:    "",
		},
		"No such bucket": {
			s3client: newMockS3Client(
				map[string][]string{},
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: "/tmp/hello-art.tar.gz",
			done:      true,
			errMsg:    "",
		},
		"No such bucket": {
			s3client: newMockS3Client(
				map[string][]string{},
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{}),
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
				map[string]interface{}{}),
			bucket:    "my-bucket",
			key:       "/folder/hello-art.tar.gz",
			localPath: tempDir,
			done:      true,
			errMsg:    "",
		},
		"Make Bucket Access Denied": {
			s3client: newMockS3Client(
				map[string][]string{},
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{
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
				map[string]interface{}{}),
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
				map[string]interface{}{}),
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
				map[string]interface{}{}),
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

func TestPutDirectoryParallel(t *testing.T) {
	// Create test directory with 1000 files
	dir, err := os.MkdirTemp("", "argo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create 1000 1KB files
	for i := 0; i < 1000; i++ {
		fpath := filepath.Join(dir, fmt.Sprintf("file-%d.txt", i))
		data := make([]byte, 1024) // 1KB
		if _, err := crand.Read(data); err != nil {
			t.Fatalf("Failed to generate random data: %v", err)
		}
		if err := os.WriteFile(fpath, data, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Create mock S3 client with parallel transfers
	workerCalls := make(map[uint64]int) // Track calls per worker
	var workerCallsMutex sync.Mutex
	workerCount := 0
	var workerCountMutex sync.Mutex
	concurrentWorkers := 0
	var concurrentWorkersMutex sync.Mutex
	maxConcurrentWorkers := 0

	client := newMockS3Client(
		map[string][]string{
			"test-bucket": {},
		},
		map[string]interface{}{
			"PutFile": func() error {
				// Get current goroutine ID to identify worker
				workerID := getGoroutineID()

				// Track unique workers
				workerCountMutex.Lock()
				if _, exists := workerCalls[workerID]; !exists {
					workerCount++
				}
				workerCountMutex.Unlock()

				// Track concurrent workers
				concurrentWorkersMutex.Lock()
				concurrentWorkers++
				if concurrentWorkers > maxConcurrentWorkers {
					maxConcurrentWorkers = concurrentWorkers
				}
				concurrentWorkersMutex.Unlock()

				// Simulate some work to ensure parallel execution
				time.Sleep(time.Millisecond)

				concurrentWorkersMutex.Lock()
				concurrentWorkers--
				concurrentWorkersMutex.Unlock()

				workerCallsMutex.Lock()
				workerCalls[workerID]++
				workerCallsMutex.Unlock()

				return nil
			},
		},
	)

	// Upload directory
	err = client.PutDirectory("test-bucket", "test", dir)
	if err != nil {
		t.Fatalf("Failed to upload directory: %v", err)
	}

	// Verify all files were uploaded
	totalCalls := 0
	workerCallsMutex.Lock()
	for _, calls := range workerCalls {
		totalCalls += calls
	}
	workerCallsMutex.Unlock()
	if totalCalls != 1000 {
		t.Errorf("Expected 1000 PutFile calls, got %d", totalCalls)
	}

	// Verify multiple workers were used
	workerCountMutex.Lock()
	wc := workerCount
	workerCountMutex.Unlock()
	if wc < 2 {
		t.Errorf("Expected multiple workers to be used, got %d", wc)
	}

	// Verify concurrent execution
	if maxConcurrentWorkers < 2 {
		t.Errorf("Expected concurrent execution with at least 2 workers, got %d", maxConcurrentWorkers)
	}

	// Verify work distribution among workers
	workerCallsMutex.Lock()
	minCalls := totalCalls
	maxCalls := 0
	for _, calls := range workerCalls {
		if calls < minCalls {
			minCalls = calls
		}
		if calls > maxCalls {
			maxCalls = calls
		}
	}
	workerCallsMutex.Unlock()

	// Verify that no worker is completely idle and no worker is overloaded
	// We expect each worker to handle at least 10% of the total work
	minExpectedCalls := totalCalls / 10
	if minCalls < minExpectedCalls {
		t.Errorf("Some workers handled too few calls: minimum was %d, expected at least %d", minCalls, minExpectedCalls)
	}

	// We expect no worker to handle more than 50% of the total work
	maxExpectedCalls := totalCalls / 2
	if maxCalls > maxExpectedCalls {
		t.Errorf("Some workers handled too many calls: maximum was %d, expected at most %d", maxCalls, maxExpectedCalls)
	}

	t.Logf("Work distributed among %d workers with maximum concurrency of %d", wc, maxConcurrentWorkers)
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

func TestParallelDownload(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "s3-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files in the temp directory
	testFiles := []string{
		"file1.txt",
		"file2.txt",
		"subdir/file3.txt",
		"subdir/file4.txt",
	}
	for _, file := range testFiles {
		path := filepath.Join(tempDir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", file, err)
		}
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Create mock S3 client with worker tracking
	concurrentWorkers := 0
	var concurrentWorkersMutex sync.Mutex
	maxConcurrentWorkers := 0

	mockClient := &mockS3Client{
		files:       make(map[string][]string),
		workerCalls: make(map[uint64]int),
		mockedErrs: map[string]interface{}{
			"GetFile": func() error {
				concurrentWorkersMutex.Lock()
				concurrentWorkers++
				if concurrentWorkers > maxConcurrentWorkers {
					maxConcurrentWorkers = concurrentWorkers
				}
				concurrentWorkersMutex.Unlock()

				// Simulate some work
				time.Sleep(time.Millisecond)

				concurrentWorkersMutex.Lock()
				concurrentWorkers--
				concurrentWorkersMutex.Unlock()

				return nil
			},
		},
	}

	// Add test files to mock S3
	bucket := "test-bucket"
	keyPrefix := "test-dir"
	for _, file := range testFiles {
		s3Key := filepath.ToSlash(filepath.Join(keyPrefix, file))
		mockClient.files[bucket] = append(mockClient.files[bucket], s3Key)
	}

	// Create download directory
	downloadDir := filepath.Join(tempDir, "download")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		t.Fatalf("Failed to create download directory: %v", err)
	}

	// Test parallel download
	err = mockClient.GetDirectory(bucket, keyPrefix, downloadDir)
	if err != nil {
		t.Fatalf("Failed to download directory: %v", err)
	}

	// Verify downloaded files
	for _, file := range testFiles {
		expectedPath := filepath.Join(downloadDir, file)
		if _, err := os.Stat(expectedPath); err != nil {
			t.Errorf("Failed to find downloaded file %s: %v", file, err)
		}
	}

	// Verify parallelism: check that multiple workers were used
	if mockClient.workerCount < 2 {
		t.Errorf("Expected multiple workers to be used, got %d", mockClient.workerCount)
	}

	// Verify concurrent execution
	if maxConcurrentWorkers < 2 {
		t.Errorf("Expected concurrent execution with at least 2 workers, got %d", maxConcurrentWorkers)
	}

	// Verify work distribution
	totalCalls := 0
	for _, calls := range mockClient.workerCalls {
		totalCalls += calls
	}

	minCalls := totalCalls
	maxCalls := 0
	for _, calls := range mockClient.workerCalls {
		if calls < minCalls {
			minCalls = calls
		}
		if calls > maxCalls {
			maxCalls = calls
		}
	}

	// Verify that no worker is completely idle and no worker is overloaded
	// We expect each worker to handle at least 25% of the total work (since we have 4 files)
	minExpectedCalls := totalCalls / 4
	if minCalls < minExpectedCalls {
		t.Errorf("Some workers handled too few calls: minimum was %d, expected at least %d", minCalls, minExpectedCalls)
	}

	// We expect no worker to handle more than 75% of the total work
	maxExpectedCalls := (totalCalls * 3) / 4
	if maxCalls > maxExpectedCalls {
		t.Errorf("Some workers handled too many calls: maximum was %d, expected at most %d", maxCalls, maxExpectedCalls)
	}

	t.Logf("Work distributed among %d workers with maximum concurrency of %d", mockClient.workerCount, maxConcurrentWorkers)
}
