package s3

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/argoproj/pkg/file"
	argos3 "github.com/argoproj/pkg/s3"
	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	artifactscommon "github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// ArtifactDriver is a driver for AWS S3
type ArtifactDriver struct {
	Endpoint              string
	Region                string
	Secure                bool
	AccessKey             string
	SecretKey             string
	RoleARN               string
	UseSDKCreds           bool
	Context               context.Context
	KmsKeyId              string
	KmsEncryptionContext  string
	EnableEncryption      bool
	ServerSideCustomerKey string
}

var (
	_            artifactscommon.ArtifactDriver = &ArtifactDriver{}
	defaultRetry                                = wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1}
)

// newMinioClient instantiates a new minio client object.
func (s3Driver *ArtifactDriver) newS3Client(ctx context.Context) (argos3.S3Client, error) {
	opts := argos3.S3ClientOpts{
		Endpoint:    s3Driver.Endpoint,
		Region:      s3Driver.Region,
		Secure:      s3Driver.Secure,
		AccessKey:   s3Driver.AccessKey,
		SecretKey:   s3Driver.SecretKey,
		RoleARN:     s3Driver.RoleARN,
		Trace:       os.Getenv(common.EnvVarArgoTrace) == "1",
		UseSDKCreds: s3Driver.UseSDKCreds,
		EncryptOpts: argos3.EncryptOpts{
			KmsKeyId:              s3Driver.KmsKeyId,
			KmsEncryptionContext:  s3Driver.KmsEncryptionContext,
			Enabled:               s3Driver.EnableEncryption,
			ServerSideCustomerKey: s3Driver.ServerSideCustomerKey,
		},
	}

	return argos3.NewS3Client(ctx, opts)
}

// Load downloads artifacts from S3 compliant storage
func (s3Driver *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			log.Infof("S3 Load path: %s, key: %s", path, inputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return !(isTransientS3Err(err) || errorsutil.IsTransientErr(err)), fmt.Errorf("failed to create new S3 client: %v", err)
			}
			return loadS3Artifact(s3cli, inputArtifact, path)
		})

	return err
}

// loadS3Artifact downloads artifacts from an S3 compliant storage
// returns true if the download is completed or can't be retried (non-transient error)
// returns false if it can be retried (transient error)
func loadS3Artifact(s3cli argos3.S3Client, inputArtifact *wfv1.Artifact, path string) (bool, error) {
	origErr := s3cli.GetFile(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path)
	if origErr == nil {
		return true, nil
	}
	if !argos3.IsS3ErrCode(origErr, "NoSuchKey") {
		return !errorsutil.IsTransientErr(origErr) || !isTransientS3Err(origErr), fmt.Errorf("failed to get file: %v", origErr)
	}
	// If we get here, the error was a NoSuchKey. The key might be a s3 "directory"
	isDir, err := s3cli.IsDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if err != nil {
		return !(isTransientS3Err(err) || errorsutil.IsTransientErr(err)), fmt.Errorf("failed to test if %s is a directory: %v", inputArtifact.S3.Key, err)
	}
	if !isDir {
		// It's neither a file, nor a directory. Return the original NoSuchKey error
		return true, errors.New(errors.CodeNotFound, origErr.Error())
	}

	if err = s3cli.GetDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path); err != nil {
		return !(isTransientS3Err(err) || errorsutil.IsTransientErr(err)), fmt.Errorf("failed to get directory: %v", err)
	}
	return true, nil
}

// Save saves an artifact to S3 compliant storage
func (s3Driver *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			log.Infof("S3 Save path: %s, key: %s", path, outputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return !(isTransientS3Err(err) || errorsutil.IsTransientErr(err)), fmt.Errorf("failed to create new S3 client: %v", err)
			}
			return saveS3Artifact(s3cli, path, outputArtifact)
		})
	return err
}

// saveS3Artifact uploads artifacts to an S3 compliant storage
// returns true if the upload is completed or can't be retried (non-transient error)
// returns false if it can be retried (transient error)
func saveS3Artifact(s3cli argos3.S3Client, path string, outputArtifact *wfv1.Artifact) (bool, error) {
	isDir, err := file.IsDirectory(path)
	if err != nil {
		return true, fmt.Errorf("failed to test if %s is a directory: %v", path, err)
	}

	createBucketIfNotPresent := outputArtifact.S3.CreateBucketIfNotPresent
	if createBucketIfNotPresent != nil {
		log.Infof("Trying to create bucket: %s", outputArtifact.S3.Bucket)
		err := s3cli.MakeBucket(outputArtifact.S3.Bucket, minio.MakeBucketOptions{
			Region:        outputArtifact.S3.Region,
			ObjectLocking: outputArtifact.S3.CreateBucketIfNotPresent.ObjectLocking,
		})
		if err != nil {
			return !(isTransientS3Err(err) || errorsutil.IsTransientErr(err)), fmt.Errorf("failed to create bucket %s: %v", outputArtifact.S3.Bucket, err)
		}
	}

	if isDir {
		if err = s3cli.PutDirectory(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path); err != nil {
			return !(isTransientS3Err(err) || errorsutil.IsTransientErr(err)), fmt.Errorf("failed to put directory: %v", err)
		}
	} else {
		if err = s3cli.PutFile(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path); err != nil {
			return !(isTransientS3Err(err) || errorsutil.IsTransientErr(err)), fmt.Errorf("failed to put file: %v", err)
		}
	}
	return true, nil
}

func (s3Driver *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var files []string
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return !(isTransientS3Err(err) || errorsutil.IsTransientErr(err)), fmt.Errorf("failed to create new S3 client: %v", err)
			}
			files, err = s3cli.ListDirectory(artifact.S3.Bucket, artifact.S3.Key)
			if err != nil {
				return !(isTransientS3Err(err) || errorsutil.IsTransientErr(err)), fmt.Errorf("failed to list directory: %v", err)
			}
			return true, nil
		})

	return files, err
}
