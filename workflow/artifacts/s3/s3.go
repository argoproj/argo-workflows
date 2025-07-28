package s3

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/minio/minio-go/v7/pkg/sse"

	"github.com/minio/minio-go/v7"
	"k8s.io/client-go/util/retry"

	argoerrs "github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	artifactscommon "github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	executorretry "github.com/argoproj/argo-workflows/v3/workflow/executor/retry"
)

const nullIAMEndpoint = ""

type S3Client interface {
	// PutFile puts a single file to a bucket at the specified key
	PutFile(bucket, key, path string) error

	// PutDirectory puts a complete directory into a bucket key prefix, with each file in the directory
	// a separate key in the bucket.
	PutDirectory(bucket, key, path string) error

	// GetFile downloads a file to a local file path
	GetFile(bucket, key, path string) error

	// OpenFile opens a file for much lower disk and memory usage that GetFile
	OpenFile(bucket, key string) (io.ReadCloser, error)

	// KeyExists checks if object exists (and if we have permission to access)
	KeyExists(bucket, key string) (bool, error)

	// Delete deletes the key from the bucket
	Delete(bucket, key string) error

	// GetDirectory downloads a directory to a local file path
	GetDirectory(bucket, key, path string) error

	// ListDirectory list the contents of a directory/bucket
	ListDirectory(bucket, keyPrefix string) ([]string, error)

	// IsDirectory tests if the key is acting like an s3 directory
	IsDirectory(bucket, key string) (bool, error)

	// BucketExists returns whether a bucket exists
	BucketExists(bucket string) (bool, error)

	// MakeBucket creates a bucket with name bucketName and options opts
	MakeBucket(bucketName string, opts minio.MakeBucketOptions) error
}

type EncryptOpts struct {
	KmsKeyID              string
	KmsEncryptionContext  string
	Enabled               bool
	ServerSideCustomerKey string
}

// AddressingStyle is a type of bucket (and also its content) addressing used by the S3 client and supported by the server
type AddressingStyle int

const (
	AutoDetectStyle AddressingStyle = iota
	PathStyle
	VirtualHostedStyle
)

type S3ClientOpts struct {
	Endpoint        string
	AddressingStyle AddressingStyle
	Region          string
	Secure          bool
	Transport       http.RoundTripper
	AccessKey       string
	SecretKey       string
	SessionToken    string
	Trace           bool
	RoleARN         string
	RoleSessionName string
	UseSDKCreds     bool
	EncryptOpts     EncryptOpts
	SendContentMd5  bool
}

type s3client struct {
	S3ClientOpts
	minioClient *minio.Client
	// nolint: containedctx
	ctx context.Context
}

var _ S3Client = &s3client{}

// ArtifactDriver is a driver for AWS S3
type ArtifactDriver struct {
	Endpoint              string
	Region                string
	Secure                bool
	TrustedCA             string
	AccessKey             string
	SecretKey             string
	SessionToken          string
	RoleARN               string
	UseSDKCreds           bool
	KmsKeyID              string
	KmsEncryptionContext  string
	EnableEncryption      bool
	ServerSideCustomerKey string
}

var _ artifactscommon.ArtifactDriver = &ArtifactDriver{}

// newS3Client instantiates a new S3 client object.
func (s3Driver *ArtifactDriver) newS3Client(ctx context.Context) (S3Client, error) {
	opts := S3ClientOpts{
		Endpoint:     s3Driver.Endpoint,
		Region:       s3Driver.Region,
		Secure:       s3Driver.Secure,
		AccessKey:    s3Driver.AccessKey,
		SecretKey:    s3Driver.SecretKey,
		SessionToken: s3Driver.SessionToken,
		RoleARN:      s3Driver.RoleARN,
		Trace:        os.Getenv(common.EnvVarArgoTrace) == "1",
		UseSDKCreds:  s3Driver.UseSDKCreds,
		EncryptOpts: EncryptOpts{
			KmsKeyID:              s3Driver.KmsKeyID,
			KmsEncryptionContext:  s3Driver.KmsEncryptionContext,
			Enabled:               s3Driver.EnableEncryption,
			ServerSideCustomerKey: s3Driver.ServerSideCustomerKey,
		},
		SendContentMd5: true,
	}

	if tr, err := GetDefaultTransport(opts); err == nil {
		if s3Driver.Secure && s3Driver.TrustedCA != "" {
			// Trust only the provided root CA
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM([]byte(s3Driver.TrustedCA))
			tr.TLSClientConfig.RootCAs = pool
		}
		opts.Transport = tr
	}

	return NewS3Client(ctx, opts)
}

// Load downloads artifacts from S3 compliant storage
func (s3Driver *ArtifactDriver) Load(ctx context.Context, inputArtifact *wfv1.Artifact, path string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	log := logging.RequireLoggerFromContext(ctx)
	err := waitutil.Backoff(executorretry.ExecutorRetry(ctx),
		func() (bool, error) {
			log.Infof(ctx, "S3 Load path: %s, key: %s", path, inputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return !isTransientS3Err(ctx, err), fmt.Errorf("failed to create new S3 client: %v", err)
			}
			return loadS3Artifact(ctx, s3cli, inputArtifact, path)
		})

	return err
}

// loadS3Artifact downloads artifacts from an S3 compliant storage
// returns true if the download is completed or can't be retried (non-transient error)
// returns false if it can be retried (transient error)
func loadS3Artifact(ctx context.Context, s3cli S3Client, inputArtifact *wfv1.Artifact, path string) (bool, error) {
	origErr := s3cli.GetFile(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path)
	if origErr == nil {
		return true, nil
	}
	if !IsS3ErrCode(origErr, "NoSuchKey") {
		return !isTransientS3Err(ctx, origErr), fmt.Errorf("failed to get file: %v", origErr)
	}
	// If we get here, the error was a NoSuchKey. The key might be an s3 "directory"
	isDir, err := s3cli.IsDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if err != nil {
		return !isTransientS3Err(ctx, err), fmt.Errorf("failed to test if %s is a directory: %v", inputArtifact.S3.Key, err)
	}
	if !isDir {
		// It's neither a file, nor a directory. Return the original NoSuchKey error
		return true, argoerrs.New(argoerrs.CodeNotFound, origErr.Error())
	}

	if err = s3cli.GetDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path); err != nil {
		return !isTransientS3Err(ctx, err), fmt.Errorf("failed to get directory: %v", err)
	}
	return true, nil
}

// OpenStream opens a stream reader for an artifact from S3 compliant storage
func (s3Driver *ArtifactDriver) OpenStream(ctx context.Context, inputArtifact *wfv1.Artifact) (io.ReadCloser, error) {
	log := logging.RequireLoggerFromContext(ctx)
	log.Infof(ctx, "S3 OpenStream: key: %s", inputArtifact.S3.Key)
	// nolint:contextcheck
	s3cli, err := s3Driver.newS3Client(log.NewBackgroundContext())
	if err != nil {
		return nil, fmt.Errorf("failed to create new S3 client: %v", err)
	}

	return streamS3Artifact(ctx, s3cli, inputArtifact)
}

func streamS3Artifact(_ context.Context, s3cli S3Client, inputArtifact *wfv1.Artifact) (io.ReadCloser, error) {
	stream, origErr := s3cli.OpenFile(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if origErr == nil {
		return stream, nil
	}
	if !IsS3ErrCode(origErr, "NoSuchKey") {
		return nil, fmt.Errorf("failed to get file: %v", origErr)
	}
	// If we get here, the error was a NoSuchKey. The key might be an s3 "directory"
	isDir, err := s3cli.IsDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to test if %s is a directory: %v", inputArtifact.S3.Key, err)
	}
	if !isDir {
		// It's neither a file, nor a directory. Return the original NoSuchKey error
		return nil, argoerrs.New(argoerrs.CodeNotFound, origErr.Error())
	}
	// directory case:
	// todo: make a .tgz file which can be streamed to user
	return nil, argoerrs.New(argoerrs.CodeNotImplemented, "Directory Stream capability currently unimplemented for S3")
}

// Save saves an artifact to S3 compliant storage
func (s3Driver *ArtifactDriver) Save(ctx context.Context, path string, outputArtifact *wfv1.Artifact) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	log := logging.RequireLoggerFromContext(ctx)
	err := waitutil.Backoff(executorretry.ExecutorRetry(ctx),
		func() (bool, error) {
			log.Infof(ctx, "S3 Save path: %s, key: %s", path, outputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return !isTransientS3Err(ctx, err), fmt.Errorf("failed to create new S3 client: %v", err)
			}
			return saveS3Artifact(ctx, s3cli, path, outputArtifact)
		})
	return err
}

// Delete deletes an artifact from an S3 compliant storage
func (s3Driver *ArtifactDriver) Delete(ctx context.Context, artifact *wfv1.Artifact) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	log := logging.RequireLoggerFromContext(ctx)
	err := retry.OnError(retry.DefaultBackoff, func(err error) bool {
		return isTransientS3Err(ctx, err)
	}, func() error {
		log.Infof(ctx, "S3 Delete artifact: key: %s", artifact.S3.Key)
		s3cli, err := s3Driver.newS3Client(ctx)
		if err != nil {
			return err
		}

		// check suffix instead of s3cli.IsDirectory as it requires another request for file delete (most scenarios)
		if !strings.HasSuffix(artifact.S3.Key, "/") {
			return s3cli.Delete(artifact.S3.Bucket, artifact.S3.Key)
		}

		keys, err := s3cli.ListDirectory(artifact.S3.Bucket, artifact.S3.Key)
		if err != nil {
			return fmt.Errorf("unable to list files in %s: %s", artifact.S3.Key, err)
		}
		for _, objKey := range keys {
			err = s3cli.Delete(artifact.S3.Bucket, objKey)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

// saveS3Artifact uploads artifacts to an S3 compliant storage
// returns true if the upload is completed or can't be retried (non-transient error)
// returns false if it can be retried (transient error)
func saveS3Artifact(ctx context.Context, s3cli S3Client, path string, outputArtifact *wfv1.Artifact) (bool, error) {
	isDir, err := file.IsDirectory(path)
	if err != nil {
		return true, fmt.Errorf("failed to test if %s is a directory: %v", path, err)
	}
	log := logging.RequireLoggerFromContext(ctx)
	createBucketIfNotPresent := outputArtifact.S3.CreateBucketIfNotPresent
	if createBucketIfNotPresent != nil {
		log.WithField("bucket", outputArtifact.S3.Bucket).Info(ctx, "creating bucket")
		err := s3cli.MakeBucket(outputArtifact.S3.Bucket, minio.MakeBucketOptions{
			Region:        outputArtifact.S3.Region,
			ObjectLocking: outputArtifact.S3.CreateBucketIfNotPresent.ObjectLocking,
		})
		alreadyExists := bucketAlreadyExistsErr(err)
		log.WithField("bucket", outputArtifact.S3.Bucket).
			WithField("alreadyExists", alreadyExists).
			WithError(err).
			Info(ctx, "create bucket failed")
		if err != nil && !alreadyExists {
			return !isTransientS3Err(ctx, err), fmt.Errorf("failed to create bucket %s: %v", outputArtifact.S3.Bucket, err)
		}
	}

	if isDir {
		if err = s3cli.PutDirectory(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path); err != nil {
			return !isTransientS3Err(ctx, err), fmt.Errorf("failed to put directory: %v", err)
		}
	} else {
		if err = s3cli.PutFile(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path); err != nil {
			return !isTransientS3Err(ctx, err), fmt.Errorf("failed to put file: %v", err)
		}
	}
	return true, nil
}

func bucketAlreadyExistsErr(err error) bool {
	resp := &minio.ErrorResponse{}
	// https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html
	alreadyExistsCodes := map[string]bool{"BucketAlreadyExists": true, "BucketAlreadyOwnedByYou": true}
	return errors.As(err, resp) && alreadyExistsCodes[resp.Code]
}

// ListObjects returns the files inside the directory represented by the Artifact
func (s3Driver *ArtifactDriver) ListObjects(ctx context.Context, artifact *wfv1.Artifact) ([]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var files []string
	var done bool
	err := waitutil.Backoff(executorretry.ExecutorRetry(ctx),
		func() (bool, error) {
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return !isTransientS3Err(ctx, err), fmt.Errorf("failed to create new S3 client: %v", err)
			}
			done, files, err = listObjects(ctx, s3cli, artifact)
			return done, err
		})

	return files, err
}

// listObjects returns the files inside the directory represented by the Artifact
// returns true if success or can't be retried (non-transient error)
// returns false if it can be retried (transient error)
func listObjects(ctx context.Context, s3cli S3Client, artifact *wfv1.Artifact) (bool, []string, error) {
	var files []string
	files, err := s3cli.ListDirectory(artifact.S3.Bucket, artifact.S3.Key)
	if err != nil {
		return !isTransientS3Err(ctx, err), files, fmt.Errorf("failed to list directory: %v", err)
	}
	log := logging.RequireLoggerFromContext(ctx)
	log.Debugf(ctx, "successfully listing S3 directory associated with bucket: %s and key %s: %v", artifact.S3.Bucket, artifact.S3.Key, files)

	if len(files) == 0 {
		directoryExists, err := s3cli.KeyExists(artifact.S3.Bucket, artifact.S3.Key)
		if err != nil {
			return !isTransientS3Err(ctx, err), files, fmt.Errorf("failed to check if key %s exists from bucket %s: %v", artifact.S3.Key, artifact.S3.Bucket, err)
		}
		if !directoryExists {
			return true, files, argoerrs.New(argoerrs.CodeNotFound, fmt.Sprintf("no key found of name %s", artifact.S3.Key))
		}
	}
	return true, files, nil
}

func (s3Driver *ArtifactDriver) IsDirectory(ctx context.Context, artifact *wfv1.Artifact) (bool, error) {
	s3cli, err := s3Driver.newS3Client(ctx)
	if err != nil {
		return false, err
	}
	return s3cli.IsDirectory(artifact.S3.Bucket, artifact.S3.Key)
}

// Get AWS credentials based on default order from aws SDK
func getAWSCredentials(ctx context.Context, opts S3ClientOpts) (*credentials.Credentials, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.Region))
	if err != nil {
		return nil, err
	}

	value, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, err
	}
	return credentials.NewStaticV4(value.AccessKeyID, value.SecretAccessKey, value.SessionToken), nil
}

// GetAssumeRoleCredentials gets Assumed role credentials
func getAssumeRoleCredentials(ctx context.Context, opts S3ClientOpts) (*credentials.Credentials, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := sts.NewFromConfig(cfg)

	// Create the credentials from AssumeRoleProvider to assume the role
	// referenced by the "myRoleARN" ARN. Prompt for MFA token from stdin.

	creds := stscreds.NewAssumeRoleProvider(client, opts.RoleARN)
	value, err := creds.Retrieve(ctx)
	if err != nil {
		return nil, err
	}
	return credentials.NewStaticV4(value.AccessKeyID, value.SecretAccessKey, value.SessionToken), nil
}

func GetCredentials(ctx context.Context, opts S3ClientOpts) (*credentials.Credentials, error) {
	log := logging.RequireLoggerFromContext(ctx)
	if opts.AccessKey != "" && opts.SecretKey != "" {
		if opts.SessionToken != "" {
			log.WithField("endpoint", opts.Endpoint).Info(ctx, "Creating minio client using ephemeral credentials")
			return credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, opts.SessionToken), nil
		} else {
			log.WithField("endpoint", opts.Endpoint).Info(ctx, "Creating minio client using static credentials")
			return credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""), nil
		}
	} else if opts.RoleARN != "" {
		log.WithField("roleArn", opts.RoleARN).Info(ctx, "Creating minio client using assumed-role credentials")
		return getAssumeRoleCredentials(ctx, opts)
	} else if opts.UseSDKCreds {
		log.Info(ctx, "Creating minio client using AWS SDK credentials")
		return getAWSCredentials(ctx, opts)
	} else {
		log.Info(ctx, "Creating minio client using IAM role")
		return credentials.NewIAM(nullIAMEndpoint), nil
	}
}

// GetDefaultTransport returns minio's default transport
func GetDefaultTransport(opts S3ClientOpts) (*http.Transport, error) {
	return minio.DefaultTransport(opts.Secure)
}

// NewS3Client instantiates a new S3 client object backed
func NewS3Client(ctx context.Context, opts S3ClientOpts) (S3Client, error) {
	ctx, _ = logging.RequireLoggerFromContext(ctx).WithField("component", "s3_client").InContext(ctx)
	s3cli := s3client{
		S3ClientOpts: opts,
	}
	s3cli.AccessKey = strings.TrimSpace(s3cli.AccessKey)
	s3cli.SecretKey = strings.TrimSpace(s3cli.SecretKey)
	var minioClient *minio.Client
	var err error

	credentials, err := GetCredentials(ctx, opts)
	if err != nil {
		return nil, err
	}

	var bucketLookupType minio.BucketLookupType
	switch s3cli.AddressingStyle {
	case PathStyle:
		bucketLookupType = minio.BucketLookupPath
	case VirtualHostedStyle:
		bucketLookupType = minio.BucketLookupDNS
	default:
		bucketLookupType = minio.BucketLookupAuto
	}
	minioOpts := &minio.Options{Creds: credentials, Secure: s3cli.Secure, Transport: opts.Transport, Region: s3cli.Region, BucketLookup: bucketLookupType}
	minioClient, err = minio.New(s3cli.Endpoint, minioOpts)
	if err != nil {
		return nil, err
	}
	if opts.Trace {
		minioClient.TraceOn(os.Stderr)
	}

	if opts.EncryptOpts.KmsKeyID != "" && opts.EncryptOpts.ServerSideCustomerKey != "" {
		return nil, fmt.Errorf("EncryptOpts.KmsKeyId and EncryptOpts.SSECPassword cannot be set together")
	}

	if opts.EncryptOpts.ServerSideCustomerKey != "" && !opts.Secure {
		return nil, fmt.Errorf("Secure must be set if EncryptOpts.SSECPassword is set")
	}

	s3cli.ctx = ctx
	s3cli.minioClient = minioClient

	return &s3cli, nil
}

// PutFile puts a single file to a bucket at the specified key
func (s *s3client) PutFile(bucket, key, path string) error {
	logging.RequireLoggerFromContext(s.ctx).WithFields(logging.Fields{"endpoint": s.Endpoint, "bucket": bucket, "key": key, "path": path}).Info(s.ctx, "Saving file to s3")
	// NOTE: minio will detect proper mime-type based on file extension

	encOpts, err := s.EncryptOpts.buildServerSideEnc(bucket, key)
	if err != nil {
		return err
	}

	_, err = s.minioClient.FPutObject(s.ctx, bucket, key, path, minio.PutObjectOptions{SendContentMd5: s.SendContentMd5, ServerSideEncryption: encOpts})
	if err != nil {
		return err
	}
	return nil
}

func (s *s3client) BucketExists(bucketName string) (bool, error) {
	logging.RequireLoggerFromContext(s.ctx).WithField("bucket", bucketName).Info(s.ctx, "Checking if bucket exists")
	result, err := s.minioClient.BucketExists(s.ctx, bucketName)
	return result, err
}

func (s *s3client) MakeBucket(bucketName string, opts minio.MakeBucketOptions) error {
	logging.RequireLoggerFromContext(s.ctx).WithFields(logging.Fields{"bucket": bucketName, "region": opts.Region, "objectLocking": opts.ObjectLocking}).Info(s.ctx, "Creating bucket")
	err := s.minioClient.MakeBucket(s.ctx, bucketName, opts)
	if err != nil {
		return err
	}

	err = s.setBucketEnc(bucketName)
	return err
}

type uploadTask struct {
	key  string
	path string
}

func generatePutTasks(ctx context.Context, keyPrefix, rootPath string) chan uploadTask {
	rootPath = filepath.Clean(rootPath) + string(os.PathSeparator)
	uploadTasks := make(chan uploadTask)
	go func() {
		log := logging.RequireLoggerFromContext(ctx)
		_ = filepath.Walk(rootPath, func(localPath string, fi os.FileInfo, err error) error {
			if err != nil {
				log.WithFields(logging.Fields{"localPath": localPath}).WithError(err).Error(ctx, "Failed to walk artifacts path")
				return err
			}
			relPath := strings.TrimPrefix(localPath, rootPath)
			if fi.IsDir() {
				return nil
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				return nil
			}
			t := uploadTask{
				key:  path.Join(keyPrefix, relPath),
				path: localPath,
			}
			uploadTasks <- t
			return nil
		})
		close(uploadTasks)
	}()
	return uploadTasks
}

// PutDirectory puts a complete directory into a bucket key prefix, with each file in the directory
// a separate key in the bucket.
func (s *s3client) PutDirectory(bucket, key, path string) error {
	for putTask := range generatePutTasks(s.ctx, key, path) {
		err := s.PutFile(bucket, putTask.key, putTask.path)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetFile downloads a file to a local file path
func (s *s3client) GetFile(bucket, key, path string) error {
	logging.RequireLoggerFromContext(s.ctx).WithFields(logging.Fields{"endpoint": s.Endpoint, "bucket": bucket, "key": key, "path": path}).Info(s.ctx, "Getting file from s3")

	encOpts, err := s.EncryptOpts.buildServerSideEnc(bucket, key)
	if err != nil {
		return err
	}

	err = s.minioClient.FGetObject(s.ctx, bucket, key, path, minio.GetObjectOptions{ServerSideEncryption: encOpts})
	if err != nil {
		return err
	}
	return nil
}

// OpenFile opens a file for reading
func (s *s3client) OpenFile(bucket, key string) (io.ReadCloser, error) {
	logging.RequireLoggerFromContext(s.ctx).WithFields(logging.Fields{"endpoint": s.Endpoint, "bucket": bucket, "key": key}).Info(s.ctx, "Opening file from s3")

	encOpts, err := s.EncryptOpts.buildServerSideEnc(bucket, key)
	if err != nil {
		return nil, err
	}
	f, err := s.minioClient.GetObject(s.ctx, bucket, key, minio.GetObjectOptions{ServerSideEncryption: encOpts})
	if err != nil {
		return nil, err
	}
	// the call above doesn't return an error in the case that the key doesn't exist, but by calling Stat() it will
	_, err = f.Stat()
	if err != nil {
		return nil, err
	}
	return f, nil
}

// checks if object exists (and if we have permission to access)
func (s *s3client) KeyExists(bucket, key string) (bool, error) {
	logging.RequireLoggerFromContext(s.ctx).WithFields(logging.Fields{"endpoint": s.Endpoint, "bucket": bucket, "key": key}).Info(s.ctx, "Checking key exists from s3")

	encOpts, err := s.EncryptOpts.buildServerSideEnc(bucket, key)
	if err != nil {
		return false, err
	}

	_, err = s.minioClient.StatObject(s.ctx, bucket, key, minio.StatObjectOptions{ServerSideEncryption: encOpts})
	if err == nil {
		return true, nil
	}
	if IsS3ErrCode(err, "NoSuchKey") {
		return false, nil
	}

	return false, err
}

func (s *s3client) Delete(bucket, key string) error {
	logging.RequireLoggerFromContext(s.ctx).WithFields(logging.Fields{"endpoint": s.Endpoint, "bucket": bucket, "key": key}).Info(s.ctx, "Deleting object from s3")
	return s.minioClient.RemoveObject(s.ctx, bucket, key, minio.RemoveObjectOptions{})
}

// GetDirectory downloads a s3 directory to a local path
func (s *s3client) GetDirectory(bucket, keyPrefix, path string) error {
	logging.RequireLoggerFromContext(s.ctx).WithFields(logging.Fields{"endpoint": s.Endpoint, "bucket": bucket, "key": keyPrefix, "path": path}).Info(s.ctx, "Getting directory from s3")
	keys, err := s.ListDirectory(bucket, keyPrefix)
	if err != nil {
		return err
	}

	for _, objKey := range keys {
		relKeyPath := strings.TrimPrefix(objKey, keyPrefix)
		localPath := filepath.Join(path, relKeyPath)

		encOpts, err := s.EncryptOpts.buildServerSideEnc(bucket, objKey)
		if err != nil {
			return err
		}

		err = s.minioClient.FGetObject(s.ctx, bucket, objKey, localPath, minio.GetObjectOptions{ServerSideEncryption: encOpts})
		if err != nil {
			return err
		}
	}
	return nil
}

// IsDirectory tests if the key is acting like a s3 directory. This just means it has at least one
// object which is prefixed with the given key
func (s *s3client) IsDirectory(bucket, keyPrefix string) (bool, error) {
	doneCh := make(chan struct{})
	defer close(doneCh)

	if keyPrefix != "" {
		keyPrefix = filepath.Clean(keyPrefix) + "/"
		if os.PathSeparator == '\\' {
			keyPrefix = strings.ReplaceAll(keyPrefix, "\\", "/")
		}
	}

	listOpts := minio.ListObjectsOptions{
		Prefix:    keyPrefix,
		Recursive: false,
	}
	objCh := s.minioClient.ListObjects(s.ctx, bucket, listOpts)
	for obj := range objCh {
		if obj.Err != nil {
			return false, obj.Err
		} else {
			return true, nil
		}
	}
	return false, nil
}

func (s *s3client) ListDirectory(bucket, keyPrefix string) ([]string, error) {
	logging.RequireLoggerFromContext(s.ctx).WithFields(logging.Fields{"endpoint": s.Endpoint, "bucket": bucket, "key": keyPrefix}).Info(s.ctx, "Listing directory from s3")

	if keyPrefix != "" {
		keyPrefix = filepath.Clean(keyPrefix) + "/"
		if os.PathSeparator == '\\' {
			keyPrefix = strings.ReplaceAll(keyPrefix, "\\", "/")
		}
	}

	doneCh := make(chan struct{})
	defer close(doneCh)
	listOpts := minio.ListObjectsOptions{
		Prefix:    keyPrefix,
		Recursive: true,
	}
	var out []string
	objCh := s.minioClient.ListObjects(s.ctx, bucket, listOpts)
	for obj := range objCh {
		if obj.Err != nil {
			return nil, obj.Err
		}
		if strings.HasSuffix(obj.Key, "/") {
			// When a dir is created through AWS S3 console, a nameless obj will be created
			// automatically, its key will be {dir_name} + "/". This obj does not display in the
			// console, but you can see it when using aws cli.
			// If obj.Key ends with "/" means it's a dir obj, we need to skip it, otherwise it
			// will be downloaded as a regular file with the same name as the dir, and it will
			// creates error when downloading the files under the dir.
			continue
		}
		out = append(out, obj.Key)
	}
	return out, nil
}

// IsS3ErrCode returns if the supplied error is of a specific S3 error code
func IsS3ErrCode(err error, code string) bool {
	var minioErr minio.ErrorResponse
	if errors.As(err, &minioErr) {
		return minioErr.Code == code
	}
	return false
}

// setBucketEnc sets the encryption options on a bucket
func (s *s3client) setBucketEnc(bucketName string) error {
	if !s.EncryptOpts.Enabled {
		return nil
	}

	var config *sse.Configuration
	if s.EncryptOpts.KmsKeyID != "" {
		config = sse.NewConfigurationSSEKMS(s.EncryptOpts.KmsKeyID)
	} else {
		config = sse.NewConfigurationSSES3()
	}

	logging.RequireLoggerFromContext(s.ctx).WithFields(logging.Fields{"KmsKeyID": s.EncryptOpts.KmsKeyID, "bucketName": bucketName}).Info(s.ctx, "Setting Bucket Encryption")
	err := s.minioClient.SetBucketEncryption(s.ctx, bucketName, config)
	return err
}

// buildServerSideEnc creates the minio encryption options when putting encrypted items in a bucket
func (e *EncryptOpts) buildServerSideEnc(bucket, key string) (encrypt.ServerSide, error) {
	if e == nil || !e.Enabled {
		return nil, nil
	}

	if e.ServerSideCustomerKey != "" {
		encryption := encrypt.DefaultPBKDF([]byte(e.ServerSideCustomerKey), []byte(bucket+key))

		return encryption, nil
	}

	if e.KmsKeyID != "" {
		encryptionCtx, err := parseKMSEncCntx(e.KmsEncryptionContext)
		if err != nil {
			return nil, fmt.Errorf("failed to parse KMS encryption context: %w", err)
		}

		if encryptionCtx == nil {
			// To overcome a limitation in Minio which checks interface{} == nil.
			kms, err := encrypt.NewSSEKMS(e.KmsKeyID, nil)
			if err != nil {
				return nil, err
			}

			return kms, nil
		}

		kms, err := encrypt.NewSSEKMS(e.KmsKeyID, encryptionCtx)
		if err != nil {
			return nil, err
		}

		return kms, nil
	}

	return encrypt.NewSSE(), nil
}

// parseKMSEncCntx validates if kmsEncCntx is a valid JSON
func parseKMSEncCntx(kmsEncCntx string) (*string, error) {
	if kmsEncCntx == "" {
		return nil, nil
	}

	jsonKMSEncryptionContext, err := json.Marshal(json.RawMessage(kmsEncCntx))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal KMS encryption context: %w", err)
	}

	parsedKMSEncryptionContext := base64.StdEncoding.EncodeToString(jsonKMSEncryptionContext)

	return &parsedKMSEncryptionContext, nil
}
