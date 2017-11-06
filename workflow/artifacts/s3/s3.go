package s3

import (
	"fmt"
	"path"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	"github.com/minio/minio-go"
)

// S3ArtifactDriver is a driver for AWS S3
type S3ArtifactDriver struct {
	AccessKey string
	SecretKey string
}

// Download artifacts from S3 compliant storage using Minio Go SDK
func (s3Driver *S3ArtifactDriver) Load(inputArtifact *wfv1.Artifact) error {
	// Todo: need to handle volumes if the container uses volumes
	// File path to save the artifact
	artPath := path.Join(common.ExecutorArtifactBaseDir, inputArtifact.Name)

	// Initialize minio client object.
	minioClient, err := minio.New(inputArtifact.S3.Endpoint, s3Driver.AccessKey, s3Driver.SecretKey, true)

	if err != nil {
		fmt.Println("Failed to initialize Minio client")
		return errors.InternalWrapError(err)
	}

	// Download the file to a local file path
	err = minioClient.FGetObject(inputArtifact.S3.Bucket, inputArtifact.S3.Key, artPath)

	if err != nil {
		fmt.Printf("Failed to download input artifact, %s\n", artPath)
		return errors.InternalWrapError(err)
	}

	fmt.Printf("Successfully download file, %s\n", artPath)
	return nil
}

func (s3Driver *S3ArtifactDriver) Save(path string, destURL string) (string, error) {

	return destURL, nil
}
