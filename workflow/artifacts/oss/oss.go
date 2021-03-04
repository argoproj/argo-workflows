package oss

import (
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/argoproj/pkg/file"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// ArtifactDriver is a driver for OSS
type ArtifactDriver struct {
	Endpoint  string
	AccessKey string
	SecretKey string
}

var _ common.ArtifactDriver = &ArtifactDriver{}

func (ossDriver *ArtifactDriver) newOSSClient() (*oss.Client, error) {
	client, err := oss.New(ossDriver.Endpoint, ossDriver.AccessKey, ossDriver.SecretKey)
	if err != nil {
		log.Warnf("Failed to create new OSS client: %v", err)
		return nil, err
	}
	return client, err
}

// Downloads artifacts from OSS compliant storage, e.g., downloading an artifact into local path
func (ossDriver *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("OSS Load path: %s, key: %s", path, inputArtifact.OSS.Key)
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				return false, err
			}
			bucketName := inputArtifact.OSS.Bucket
			bucket, err := osscli.Bucket(bucketName)
			if err != nil {
				return false, err
			}
			objectName := inputArtifact.OSS.Key
			origErr := bucket.GetObjectToFile(objectName, path)
			if origErr == nil {
				return true, nil
			}
			if ossErr, ok := origErr.(oss.ServiceError); ok {
				if ossErr.Code != "NoSuchKey" {
					log.Warnf("Failed to get file: %v", origErr)
					return false, nil
				}
			}
			// If we get here, the error was a NoSuchKey. The key might be a directory.
			// There is only one method in OSS for downloading objects that does not differentiate between a file
			// and a directory so we append the a trailing slash here to differentiate that prior to downloading.
			err = bucket.GetObjectToFile(objectName+"/", path)
			if err != nil {
				return false, err
			}
			return true, nil
		})
	return err
}

// Saves an artifact to OSS compliant storage, e.g., uploading a local file to OSS bucket
func (ossDriver *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("OSS Save path: %s, key: %s", path, outputArtifact.OSS.Key)
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				log.Warnf("Failed to create new OSS client: %v", err)
				return false, nil
			}
			bucketName := outputArtifact.OSS.Bucket
			if outputArtifact.OSS.CreateBucketIfNotPresent {
				exists, err := osscli.IsBucketExist(bucketName)
				if err != nil {
					return false, fmt.Errorf("failed to check if bucket %s exists: %w", bucketName, err)
				}
				if !exists {
					err = osscli.CreateBucket(bucketName)
					if err != nil {
						log.Warnf("failed to automatically create bucket %s when it's not present: %v", bucketName, err)
					}
				}
			}
			bucket, err := osscli.Bucket(bucketName)
			if err != nil {
				return false, err
			}
			objectName := outputArtifact.OSS.Key
			isDir, err := file.IsDirectory(path)
			if err != nil {
				log.Warnf("Failed to test if %s is a directory: %v", path, err)
				return false, nil
			}
			// There is only one method in OSS for uploading objects that does not differentiate between a file and a directory
			// so we append the a trailing slash here to differentiate that prior to uploading.
			if isDir && !strings.HasSuffix(objectName, "/") {
				objectName += "/"
			}
			err = bucket.PutObjectFromFile(objectName, path)
			if err != nil {
				return false, err
			}
			return true, nil
		})
	return err
}

func (ossDriver *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	var files []string
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				return false, err
			}
			bucket, err := osscli.Bucket(artifact.OSS.Bucket)
			if err != nil {
				return false, err
			}
			results, err := bucket.ListObjects(oss.Prefix(artifact.OSS.Key))
			if err != nil {
				return false, err
			}
			for _, object := range results.Objects {
				files = append(files, object.Key)
			}
			return true, nil
		})
	return files, err
}
