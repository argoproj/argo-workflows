package s3

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	argos3 "github.com/argoproj/pkg/s3"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

const transientEnvVarKey = "TRANSIENT_ERROR_PATTERN"

type mockS3Client struct {
	// files is a map where key is bucket name and value consists of file keys
	files map[string][]string
	// mockedErrs is a map where key is the function name and value is the mocked error of that function
	mockedErrs map[string]error
}

func newMockS3Client(files map[string][]string, mockedErrs map[string]error) argos3.S3Client {
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
		s3client  argos3.S3Client
		bucket    string
		key       string
		localPath string
		errMsg    string
	}{
		"Success": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": []string{
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
					"my-bucket": []string{
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
					"my-bucket": []string{
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
					"my-bucket": []string{
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
				assert.Nil(t, err)
				assert.NotNil(t, stream)
			} else {
				assert.NotNil(t, err)
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
		s3client  argos3.S3Client
		bucket    string
		key       string
		localPath string
		done      bool
		errMsg    string
	}{
		"Success": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": []string{
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
					"my-bucket": []string{
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
					"my-bucket": []string{
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
					"my-bucket": []string{
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
					"my-bucket": []string{
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
					"my-bucket": []string{
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
				assert.Equal(t, tc.errMsg, "")
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
		s3client  argos3.S3Client
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
					"my-bucket": []string{},
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
				assert.Equal(t, tc.errMsg, "")
			}
		})
	}
}

func TestListObjects(t *testing.T) {

	tests := map[string]struct {
		s3client         argos3.S3Client
		bucket           string
		key              string
		expectedSuccess  bool
		expectedErrMsg   string
		expectedNumFiles int
	}{
		"Found objects": {
			s3client: newMockS3Client(
				map[string][]string{
					"my-bucket": []string{
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
					"my-bucket": []string{
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
					"my-bucket": []string{
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
				assert.Nil(t, err)
				assert.Equal(t, tc.expectedNumFiles, len(files))
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expectedErrMsg, err.Error())
			}
		})
	}
}
