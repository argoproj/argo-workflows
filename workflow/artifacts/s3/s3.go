package s3

import (
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/pkg/file"
	argos3 "github.com/argoproj/pkg/s3"
)

// S3ArtifactDriver is a driver for AWS S3
type S3ArtifactDriver struct {
	Endpoint  string
	Region    string
	Secure    bool
	AccessKey string
	SecretKey string
}

// newMinioClient instantiates a new minio client object.
func (s3Driver *S3ArtifactDriver) newS3Client() (argos3.S3Client, error) {
	opts := argos3.S3ClientOpts{
		Endpoint:  s3Driver.Endpoint,
		Region:    s3Driver.Region,
		Secure:    s3Driver.Secure,
		AccessKey: s3Driver.AccessKey,
		SecretKey: s3Driver.SecretKey,
	}
	return argos3.NewS3Client(opts)
}

// Load downloads artifacts from S3 compliant storage
func (s3Driver *S3ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("S3 Load path: %s, key: %s", path, inputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client()
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
				return false, origErr
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
func (s3Driver *S3ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("S3 Save path: %s, key: %s", path, outputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client()
			if err != nil {
				log.Warnf("Failed to create new S3 client: %v", err)
				return false, nil
			}
			isDir, err := file.IsDirectory(path)
			if err != nil {
				log.Warnf("Failed to test if %s is a directory: %v", path, err)
				return false, nil
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
