package s3

import (
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
	s3cli, err := s3Driver.newS3Client()
	if err != nil {
		return err
	}
	return s3cli.GetFile(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path)
}

// Save saves an artifact to S3 compliant storage
func (s3Driver *S3ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	s3cli, err := s3Driver.newS3Client()
	if err != nil {
		return err
	}
	isDir, err := file.IsDirectory(path)
	if err != nil {
		return err
	}
	if isDir {
		return s3cli.PutDirectory(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path)
	}
	return s3cli.PutFile(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path)
}
