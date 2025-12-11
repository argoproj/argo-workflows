package s3

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

// parallelMockS3Client is a mock implementation of S3Client interface for testing parallel operations
type parallelMockS3Client struct {
	// files is a map where key is bucket name and value consists of file keys
	files map[string][]string
	// mockedErrs is a map where key is the function name and value is the mocked error of that function
	mockedErrs map[string]error
	// parallelism configuration
	enableParallelism  bool
	fileCountThreshold int32
	parallelism        int32
}

func newParallelMockS3Client(files map[string][]string, mockedErrs map[string]error) S3Client {
	return &parallelMockS3Client{
		files:              files,
		mockedErrs:         mockedErrs,
		enableParallelism:  true,
		fileCountThreshold: 5,
		parallelism:        3,
	}
}

func (s *parallelMockS3Client) getMockedErr(funcName string) error {
	err, ok := s.mockedErrs[funcName]
	if !ok {
		return nil
	}
	return err
}

// PutFile puts a single file to a bucket at the specified key
func (s *parallelMockS3Client) PutFile(bucket, key, path string) error {
	return s.getMockedErr("PutFile")
}

// PutDirectory puts a complete directory into a bucket key prefix
func (s *parallelMockS3Client) PutDirectory(bucket, key, path string) error {
	if err := s.getMockedErr("PutDirectory"); err != nil {
		return err
	}

	// Simulate parallel upload by checking if we have enough files
	fileCount := 0
	for range generatePutTasksForTest(key, path) {
		fileCount++
	}

	if s.enableParallelism && fileCount >= int(s.fileCountThreshold) {
		// Simulate parallel upload
		return s.parallelPutDirectory(bucket, key, path)
	}

	// Simulate sequential upload
	for putTask := range generatePutTasksForTest(key, path) {
		if err := s.PutFile(bucket, putTask.key, putTask.path); err != nil {
			return err
		}
	}
	return nil
}

// GetFile downloads a file to a local file path
func (s *parallelMockS3Client) GetFile(bucket, key, path string) error {
	return s.getMockedErr("GetFile")
}

// OpenFile opens a file for reading
func (s *parallelMockS3Client) OpenFile(bucket, key string) (io.ReadCloser, error) {
	return nil, s.getMockedErr("OpenFile")
}

// KeyExists checks if object exists
func (s *parallelMockS3Client) KeyExists(bucket, key string) (bool, error) {
	err := s.getMockedErr("KeyExists")
	if files, ok := s.files[bucket]; ok {
		for _, file := range files {
			if file == key {
				return true, err
			}
		}
	}
	return false, err
}

// Delete deletes the key from the bucket
func (s *parallelMockS3Client) Delete(bucket, key string) error {
	return s.getMockedErr("Delete")
}

// GetDirectory downloads a directory to a local file path
func (s *parallelMockS3Client) GetDirectory(bucket, key, path string) error {
	if err := s.getMockedErr("GetDirectory"); err != nil {
		return err
	}

	keys, err := s.ListDirectory(bucket, key)
	if err != nil {
		return err
	}

	if s.enableParallelism && len(keys) >= int(s.fileCountThreshold) {
		// Simulate parallel download
		return s.parallelGetDirectory(bucket, key, path, keys)
	}

	// Simulate sequential download
	for _, objKey := range keys {
		relKeyPath := strings.TrimPrefix(objKey, key)
		localPath := filepath.Join(path, relKeyPath)
		if err := s.GetFile(bucket, objKey, localPath); err != nil {
			return err
		}
	}
	return nil
}

// ListDirectory list the contents of a directory/bucket
func (s *parallelMockS3Client) ListDirectory(bucket, keyPrefix string) ([]string, error) {
	err := s.getMockedErr("ListDirectory")
	if files, ok := s.files[bucket]; ok {
		return files, err
	}
	return nil, err
}

// IsDirectory tests if the key is acting like an s3 directory
func (s *parallelMockS3Client) IsDirectory(bucket, key string) (bool, error) {
	err := s.getMockedErr("IsDirectory")
	if files, ok := s.files[bucket]; ok {
		for _, file := range files {
			if file == key+"/" {
				return true, err
			}
		}
	}
	return false, err
}

// BucketExists returns whether a bucket exists
func (s *parallelMockS3Client) BucketExists(bucket string) (bool, error) {
	err := s.getMockedErr("BucketExists")
	_, exists := s.files[bucket]
	return exists, err
}

// MakeBucket creates a bucket with name bucketName and options opts
func (s *parallelMockS3Client) MakeBucket(bucketName string, opts minio.MakeBucketOptions) error {
	return s.getMockedErr("MakeBucket")
}

func (s *parallelMockS3Client) parallelPutDirectory(bucket, key, path string) error {
	tasks := generatePutTasksForTest(key, path)
	errors := make(chan error, s.parallelism)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < int(s.parallelism); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				err := s.PutFile(bucket, task.key, task.path)
				if err != nil {
					errors <- err
					return
				}
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *parallelMockS3Client) parallelGetDirectory(bucket, keyPrefix, path string, keys []string) error {
	tasks := generateGetTasks(keyPrefix, path, keys)
	errors := make(chan error, s.parallelism)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < int(s.parallelism); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				err := s.GetFile(bucket, task.key, task.path)
				if err != nil {
					errors <- err
					return
				}
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

func TestParallelPutDirectory(t *testing.T) {
	tests := map[string]struct {
		enableParallelism  bool
		fileCountThreshold int32
		parallelism        int32
		expectedErr        error
	}{
		"Parallel upload enabled": {
			enableParallelism:  true,
			fileCountThreshold: 5,
			parallelism:        3,
			expectedErr:        nil,
		},
		"Parallel upload disabled": {
			enableParallelism:  false,
			fileCountThreshold: 5,
			parallelism:        3,
			expectedErr:        nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a temporary directory for test files
			tempDir := t.TempDir()

			// Create some test files
			for i := 0; i < 10; i++ {
				filePath := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
				err := os.WriteFile(filePath, []byte("test content"), 0644)
				require.NoError(t, err)
			}

			// Create mock client
			mockClient := newParallelMockS3Client(map[string][]string{}, map[string]error{})
			mockClientImpl := mockClient.(*parallelMockS3Client)
			mockClientImpl.enableParallelism = tc.enableParallelism
			mockClientImpl.fileCountThreshold = tc.fileCountThreshold
			mockClientImpl.parallelism = tc.parallelism

			// Test PutDirectory
			err := mockClient.PutDirectory("test-bucket", "test-key", tempDir)
			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParallelGetDirectory(t *testing.T) {
	tests := map[string]struct {
		enableParallelism  bool
		fileCountThreshold int32
		parallelism        int32
		expectedErr        error
	}{
		"Parallel download enabled": {
			enableParallelism:  true,
			fileCountThreshold: 5,
			parallelism:        3,
			expectedErr:        nil,
		},
		"Parallel download disabled": {
			enableParallelism:  false,
			fileCountThreshold: 5,
			parallelism:        3,
			expectedErr:        nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a temporary directory for downloads
			tempDir := t.TempDir()

			// Create mock client with some test files
			files := map[string][]string{
				"test-bucket": {
					"test-key/file1.txt",
					"test-key/file2.txt",
					"test-key/file3.txt",
					"test-key/file4.txt",
					"test-key/file5.txt",
					"test-key/file6.txt",
				},
			}
			mockClient := newParallelMockS3Client(files, map[string]error{})
			mockClientImpl := mockClient.(*parallelMockS3Client)
			mockClientImpl.enableParallelism = tc.enableParallelism
			mockClientImpl.fileCountThreshold = tc.fileCountThreshold
			mockClientImpl.parallelism = tc.parallelism

			// Test GetDirectory
			err := mockClient.GetDirectory("test-bucket", "test-key", tempDir)
			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParallelConfig(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// Create a new S3 client with parallel configuration
	s3cli, err := NewS3Client(ctx, S3ClientOpts{
		Endpoint:           "test-endpoint",
		EnableParallelism:  boolPtr(true),
		FileCountThreshold: int32Ptr(5),
		Parallelism:        int32Ptr(3),
	})
	require.NoError(t, err)

	// Verify the configuration
	assert.True(t, *s3cli.(*s3client).EnableParallelism)
	assert.Equal(t, int32(5), *s3cli.(*s3client).FileCountThreshold)
	assert.Equal(t, int32(3), *s3cli.(*s3client).Parallelism)
}

// Helper functions for creating pointers
func boolPtr(b bool) *bool {
	return &b
}

func int32Ptr(i int32) *int32 {
	return &i
}

type putTask struct {
	key  string
	path string
}

type getTask struct {
	key  string
	path string
}

func generatePutTasksForTest(keyPrefix, path string) chan putTask {
	tasks := make(chan putTask)
	go func() {
		defer close(tasks)
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(path, filePath)
				if err != nil {
					return err
				}
				key := filepath.Join(keyPrefix, relPath)
				tasks <- putTask{key: key, path: filePath}
			}
			return nil
		})
		if err != nil {
			// In a real implementation, we would handle this error
			// For the mock, we'll just log it
			fmt.Printf("Error walking directory: %v\n", err)
		}
	}()
	return tasks
}

func generateGetTasks(keyPrefix, path string, keys []string) chan getTask {
	tasks := make(chan getTask)
	go func() {
		defer close(tasks)
		for _, key := range keys {
			relKeyPath := strings.TrimPrefix(key, keyPrefix)
			localPath := filepath.Join(path, relKeyPath)
			tasks <- getTask{key: key, path: localPath}
		}
	}()
	return tasks
}
