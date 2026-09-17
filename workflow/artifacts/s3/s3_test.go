package s3

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

const transientEnvVarKey = "TRANSIENT_ERROR_PATTERN"

type mockClient struct {
	// files is a map where key is bucket name and value consists of file keys
	files map[string][]string
	// mockedErrs is a map where key is the function name and value is the mocked error of that function
	mockedErrs map[string]error
}

func newMockClient(files map[string][]string, mockedErrs map[string]error) Client {
	return &mockClient{
		files:      files,
		mockedErrs: mockedErrs,
	}
}

func (s *mockClient) getMockedErr(funcName string) error {
	err, ok := s.mockedErrs[funcName]
	if !ok {
		return nil
	}
	return err
}

// PutFile puts a single file to a bucket at the specified key
func (s *mockClient) PutFile(bucket, key, path string) error {
	return s.getMockedErr("PutFile")
}

// PutDirectory puts a complete directory into a bucket key prefix, with each file in the directory
// a separate key in the bucket.
func (s *mockClient) PutDirectory(bucket, key, path string) error {
	return s.getMockedErr("PutDirectory")
}

// GetFile downloads a file to a local file path
func (s *mockClient) GetFile(bucket, key, path string) error {
	return s.getMockedErr("GetFile")
}

func (s *mockClient) OpenFile(bucket, key string) (io.ReadCloser, error) {
	err := s.getMockedErr("OpenFile")
	if err == nil {
		return io.NopCloser(&bytes.Buffer{}), nil
	}
	return nil, err
}

func (s *mockClient) KeyExists(bucket, key string) (bool, error) {
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
func (s *mockClient) GetDirectory(bucket, key, path string) error {
	return s.getMockedErr("GetDirectory")
}

// ListDirectory list the contents of a directory/bucket
func (s *mockClient) ListDirectory(bucket, keyPrefix string) ([]string, error) {
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
func (s *mockClient) IsDirectory(bucket, key string) (bool, error) {
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
func (s *mockClient) BucketExists(bucket string) (bool, error) {
	err := s.getMockedErr("BucketExists")
	if _, ok := s.files[bucket]; ok {
		return true, err
	}
	return false, err
}

// MakeBucket creates a bucket with name bucketName and options opts
func (s *mockClient) MakeBucket(bucketName string, opts minio.MakeBucketOptions) error {
	return s.getMockedErr("MakeBucket")
}

func TestOpenStreamS3Artifact(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	tests := map[string]struct {
		s3client  Client
		bucket    string
		key       string
		localPath string
		errMsg    string
	}{
		"Success": {
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			stream, err := streamS3Artifact(ctx, tc.s3client, &wfv1.Artifact{
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
func (s *mockClient) Delete(bucket, key string) error {
	return s.getMockedErr("Delete")
}

func TestLoadS3Artifact(t *testing.T) {
	tests := map[string]struct {
		s3client  Client
		bucket    string
		key       string
		localPath string
		done      bool
		errMsg    string
	}{
		"Success": {
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			ctx := logging.TestContext(t.Context())
			success, err := loadS3Artifact(ctx, tc.s3client, &wfv1.Artifact{
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
	ctx := logging.TestContext(t.Context())

	tempDir := t.TempDir()

	tempFile := filepath.Join(tempDir, "tmpfile")
	if err := os.WriteFile(tempFile, []byte("temporary file's content"), 0o600); err != nil {
		panic(err)
	}

	tests := map[string]struct {
		s3client  Client
		bucket    string
		key       string
		localPath string
		// skipBucketCreation leaves CreateBucketIfNotPresent nil, covering the
		// path where MakeBucket must not be called at all.
		skipBucketCreation bool
		done               bool
		errMsg             string
	}{
		"Skips MakeBucket when CreateBucketIfNotPresent is unset": {
			s3client: newMockClient(
				map[string][]string{
					"my-bucket": {},
				},
				map[string]error{
					// If MakeBucket were called, this would fail the upload.
					"MakeBucket": minio.ErrorResponse{
						Code: "AccessDenied",
					},
				}),
			bucket:             "my-bucket",
			key:                "/folder/hello-art.tar.gz",
			localPath:          tempFile,
			skipBucketCreation: true,
			done:               true,
			errMsg:             "",
		},
		"Success as File": {
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			var createBucketIfNotPresent *wfv1.CreateS3BucketOptions
			if !tc.skipBucketCreation {
				createBucketIfNotPresent = &wfv1.CreateS3BucketOptions{}
			}
			success, err := saveS3Artifact(ctx,
				tc.s3client,
				tc.localPath,
				&wfv1.Artifact{
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							S3Bucket: wfv1.S3Bucket{
								Bucket:                   tc.bucket,
								CreateBucketIfNotPresent: createBucketIfNotPresent,
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
	ctx := logging.TestContext(t.Context())
	tests := map[string]struct {
		s3client         Client
		bucket           string
		key              string
		expectedSuccess  bool
		expectedErrMsg   string
		expectedNumFiles int
	}{
		"Found objects": {
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			s3client: newMockClient(
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
			_, files, err := listObjects(ctx, tc.s3client,
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

// TestNewClient tests the s3 constructor
func TestNewClient(t *testing.T) {
	opts := ClientOpts{
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
	ctx := logging.TestContext(t.Context())
	s3If, err := NewClient(ctx, opts)
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
	assert.Nil(t, s3cli.TokenExpiration)
	// s3cli.minioClient.
	// 	s3client.minioClient
}

// TestNewClient tests the S3 constructor using ephemeral credentials
func TestNewClientEphemeral(t *testing.T) {
	opts := ClientOpts{
		Endpoint:     "foo.com",
		Region:       "us-south-3",
		AccessKey:    "key",
		SecretKey:    "secret",
		SessionToken: "sessionToken",
	}
	ctx := logging.TestContext(t.Context())
	s3If, err := NewClient(ctx, opts)
	require.NoError(t, err)
	s3cli := s3If.(*s3client)
	assert.Equal(t, opts.Endpoint, s3cli.Endpoint)
	assert.Equal(t, opts.Region, s3cli.Region)
	assert.Equal(t, opts.AccessKey, s3cli.AccessKey)
	assert.Equal(t, opts.SecretKey, s3cli.SecretKey)
	assert.Equal(t, opts.SessionToken, s3cli.SessionToken)
}

// TestNewClient tests the s3 constructor
func TestNewClientWithDiff(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	t.Run("IAMRole", func(t *testing.T) {
		opts := ClientOpts{
			Endpoint: "foo.com",
			Region:   "us-south-3",
			Secure:   false,
			Trace:    true,
		}
		s3If, err := NewClient(ctx, opts)
		require.NoError(t, err)
		s3cli := s3If.(*s3client)
		assert.Equal(t, opts.Endpoint, s3cli.Endpoint)
		assert.Equal(t, opts.Region, s3cli.Region)
		assert.Equal(t, opts.Trace, s3cli.Trace)
		assert.Equal(t, opts.Endpoint, s3cli.minioClient.EndpointURL().Host)
	})
	t.Run("AssumeIAMRole", func(t *testing.T) {
		t.SkipNow()
		opts := ClientOpts{
			Endpoint: "foo.com",
			Region:   "us-south-3",
			Secure:   false,
			Trace:    true,
			RoleARN:  "01234567890123456789",
		}
		s3If, err := NewClient(ctx, opts)
		require.NoError(t, err)
		s3cli := s3If.(*s3client)
		assert.Equal(t, opts.Endpoint, s3cli.Endpoint)
		assert.Equal(t, opts.Region, s3cli.Region)
		assert.Equal(t, opts.Trace, s3cli.Trace)
		assert.Equal(t, opts.Endpoint, s3cli.minioClient.EndpointURL().Host)
	})
}

func TestDisallowedComboOptions(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	t.Run("KMS and SSEC", func(t *testing.T) {
		opts := ClientOpts{
			Endpoint:    "foo.com",
			Region:      "us-south-3",
			Secure:      true,
			Trace:       true,
			EncryptOpts: EncryptOpts{Enabled: true, ServerSideCustomerKey: "PASSWORD", KmsKeyID: "00000000-0000-0000-0000-000000000000", KmsEncryptionContext: ""},
		}
		_, err := NewClient(ctx, opts)
		assert.Error(t, err)
	})

	t.Run("SSEC and InSecure", func(t *testing.T) {
		opts := ClientOpts{
			Endpoint:    "foo.com",
			Region:      "us-south-3",
			Secure:      false,
			Trace:       true,
			EncryptOpts: EncryptOpts{Enabled: true, ServerSideCustomerKey: "PASSWORD", KmsKeyID: "", KmsEncryptionContext: ""},
		}
		_, err := NewClient(ctx, opts)
		assert.Error(t, err)
	})
}

func TestGetCredentialsTokenExpirationBounds(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	roleARN := "arn:aws:iam::123456789012:role/test"

	tests := []struct {
		name          string
		expiration    *time.Duration
		wantBoundsErr bool
	}{
		{
			name:          "nil – uses STS default, no validation error",
			expiration:    nil,
			wantBoundsErr: false,
		},
		{
			name:          "below minimum (14m)",
			expiration:    new(14 * time.Minute),
			wantBoundsErr: true,
		},
		{
			name:          "zero duration",
			expiration:    new(time.Duration(0)),
			wantBoundsErr: true,
		},
		{
			name:          "exactly minimum (15m) – valid",
			expiration:    new(minTokenExpiration),
			wantBoundsErr: false,
		},
		{
			name:          "valid mid-range (1h)",
			expiration:    new(1 * time.Hour),
			wantBoundsErr: false,
		},
		{
			name:          "exactly maximum (12h) – valid",
			expiration:    new(maxTokenExpiration),
			wantBoundsErr: false,
		},
		{
			name:          "above maximum (13h)",
			expiration:    new(13 * time.Hour),
			wantBoundsErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := ClientOpts{
				RoleARN:         roleARN,
				TokenExpiration: tc.expiration,
			}
			_, err := GetCredentials(ctx, opts)
			if tc.wantBoundsErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "outside the AWS STS allowed range")
			} else if err != nil {
				// Bounds check passed; the call will fail because there are no real AWS
				// credentials in the test environment – that is expected.
				assert.NotContains(t, err.Error(), "outside the AWS STS allowed range")
			}
		})
	}
}

// TestIsWebIdentityConfigured verifies detection of web identity environment variables.
func TestIsWebIdentityConfigured(t *testing.T) {
	tests := []struct {
		name       string
		roleARN    string
		tokenFile  string
		wantResult bool
	}{
		{"both set", "arn:aws:iam::123:role/r", "/var/token", true},
		{"only role ARN", "arn:aws:iam::123:role/r", "", false},
		{"only token file", "", "/var/token", false},
		{"neither set", "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envWebIdentityRoleARN, tc.roleARN)
			t.Setenv(envWebIdentityTokenFile, tc.tokenFile)
			assert.Equal(t, tc.wantResult, isWebIdentityConfigured())
		})
	}
}

// TestGetCredentialsWebIdentityTokenExpirationBounds verifies that the same
// [15m, 12h] bounds check applies to the UseSDKCreds + web identity path.
func TestGetCredentialsWebIdentityTokenExpirationBounds(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Set the web identity env vars so the web identity path is taken.
	// The token file does not need to exist for the bounds check (which happens before
	// any network or file I/O).
	t.Setenv(envWebIdentityRoleARN, "arn:aws:iam::123456789012:role/test")
	t.Setenv(envWebIdentityTokenFile, "/tmp/nonexistent-token")

	tests := []struct {
		name          string
		expiration    *time.Duration
		wantBoundsErr bool
	}{
		{
			name:          "nil – no validation error",
			expiration:    nil,
			wantBoundsErr: false,
		},
		{
			name:          "below minimum (14m)",
			expiration:    new(14 * time.Minute),
			wantBoundsErr: true,
		},
		{
			name:          "exactly minimum (15m) – valid",
			expiration:    new(minTokenExpiration),
			wantBoundsErr: false,
		},
		{
			name:          "valid mid-range (1h)",
			expiration:    new(1 * time.Hour),
			wantBoundsErr: false,
		},
		{
			name:          "exactly maximum (12h) – valid",
			expiration:    new(maxTokenExpiration),
			wantBoundsErr: false,
		},
		{
			name:          "above maximum (13h)",
			expiration:    new(13 * time.Hour),
			wantBoundsErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := ClientOpts{
				UseSDKCreds:     true,
				TokenExpiration: tc.expiration,
			}
			_, err := GetCredentials(ctx, opts)
			if tc.wantBoundsErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "outside the AWS STS allowed range")
			} else if err != nil {
				// Bounds check passed; the STS call itself will fail (no real token file /
				// no real AWS), but not with a bounds error.
				assert.NotContains(t, err.Error(), "outside the AWS STS allowed range")
			}
		})
	}
}

// TestGetCredentialsWebIdentityFallthrough verifies that when web identity env vars
// are absent, UseSDKCreds falls through to the regular SDK credential chain.
func TestGetCredentialsWebIdentityFallthrough(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Ensure web identity vars are NOT set.
	t.Setenv(envWebIdentityRoleARN, "")
	t.Setenv(envWebIdentityTokenFile, "")

	opts := ClientOpts{
		UseSDKCreds:     true,
		TokenExpiration: new(14 * time.Minute), // out-of-range but should NOT be validated
	}
	// No bounds error expected because the web identity path is not taken.
	_, err := GetCredentials(ctx, opts)
	if err != nil {
		assert.NotContains(t, err.Error(), "outside the AWS STS allowed range")
	}
}

func TestParseAddressingStyle(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected AddressingStyle
	}{
		"empty string returns auto-detect":  {input: "", expected: AutoDetectStyle},
		"path returns path style":           {input: "path", expected: PathStyle},
		"virtual-hosted returns virtual":    {input: "virtual-hosted", expected: VirtualHostedStyle},
		"unknown value returns auto-detect": {input: "garbage", expected: AutoDetectStyle},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseAddressingStyle(tc.input))
		})
	}
}
