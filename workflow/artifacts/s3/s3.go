package s3

import (
	"context"
	"os"
	"time"

	"github.com/argoproj/pkg/file"
	argos3 "github.com/argoproj/pkg/s3"
	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	artifactscommon "github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// ArtifactDriver is a driver for AWS S3
type ArtifactDriver struct {
	Endpoint    string
	Region      string
	Secure      bool
	AccessKey   string
	SecretKey   string
	RoleARN     string
	UseSDKCreds bool
	Context     context.Context
}

var _ artifactscommon.ArtifactDriver = &ArtifactDriver{}

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
	}
	return argos3.NewS3Client(ctx, opts)
}

// Load downloads artifacts from S3 compliant storage
func (s3Driver *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("S3 Load path: %s, key: %s", path, inputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				log.Warnf("Failed to create new S3 client: %v", err)
				return false, nil
			}
			origErr := s3cli.GetFile(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path)
			if origErr == nil {
				return true, nil
			}
			if !argos3.IsS3ErrCode(origErr, "NoSuchKey") {
				log.Warnf("Failed get file: %v", origErr)
				return false, nil
			}
			// If we get here, the error was a NoSuchKey. The key might be a s3 "directory"
			isDir, err := s3cli.IsDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
			if err != nil {
				log.Warnf("Failed to test if %s is a directory: %v", inputArtifact.S3.Bucket, err)
				return false, nil
			}
			if !isDir {
				// It's neither a file, nor a directory. Return the original NoSuchKey error
				return false, errors.New(errors.CodeNotFound, origErr.Error())
			}

			if err = s3cli.GetDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path); err != nil {
				log.Warnf("Failed get directory: %v", err)
				return false, nil
			}
			return true, nil
		})

	return err
}

// Save saves an artifact to S3 compliant storage
func (s3Driver *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("S3 Save path: %s, key: %s", path, outputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				log.Warnf("Failed to create new S3 client: %v", err)
				return false, nil
			}
			isDir, err := file.IsDirectory(path)
			if err != nil {
				log.Warnf("Failed to test if %s is a directory: %v", path, err)
				return false, nil
			}

			createBucketIfNotPresent := outputArtifact.S3.CreateBucketIfNotPresent
			if createBucketIfNotPresent != nil {
				log.Infof("Trying to create bucket: %s", outputArtifact.S3.Bucket)
				err := s3cli.MakeBucket(outputArtifact.S3.Bucket, minio.MakeBucketOptions{
					Region:        outputArtifact.S3.Region,
					ObjectLocking: outputArtifact.S3.CreateBucketIfNotPresent.ObjectLocking,
				})
				if err != nil {
					log.Warnf("Failed to create bucket: %v. Error: %v", outputArtifact.S3.Bucket, err)
				}
			}

			if isDir {
				if err = s3cli.PutDirectory(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path); err != nil {
					log.Warnf("Failed to put directory: %v", err)
					return false, nil
				}
			} else {
				if err = s3cli.PutFile(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path); err != nil {
					log.Warnf("Failed to put file: %v", err)
					return false, nil
				}
			}
			return true, nil
		})
	return err
}

func (s3Driver *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var files []string
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return false, err
			}
			files, err = s3cli.ListDirectory(artifact.S3.Bucket, artifact.S3.Key)
			if err != nil {
				return false, err
			}
			return true, nil
		})

	return files, err
}
