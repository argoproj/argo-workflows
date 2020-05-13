package oss

import (
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// OSSArtifactDriver is a driver for OSS
type OSSArtifactDriver struct {
	Endpoint  string
	AccessKey string
	SecretKey string
}

func (ossDriver *OSSArtifactDriver) newOSSClient() (*oss.Client, error) {
	client, err := oss.New(ossDriver.Endpoint, ossDriver.AccessKey, ossDriver.SecretKey)
	if err != nil {
		log.Warnf("Failed to create new OSS client: %v", err)
		return nil, err
	}
	return client, err
}

// Downloads artifacts from OSS compliant storage, e.g., downloading an artifact into local path
func (ossDriver *OSSArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
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
			err = bucket.GetObjectToFile(objectName, path)
			if err != nil {
				return false, err
			}
			return true, nil
		})
	return err
}

// Saves an artifact to OSS compliant storage, e.g., uploading a local file to OSS bucket
func (ossDriver *OSSArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("OSS Save path: %s, key: %s", path, outputArtifact.OSS.Key)
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				log.Warnf("Failed to create new OSS client: %v", err)
				return false, nil
			}
			bucketName := outputArtifact.OSS.Bucket
			bucket, err := osscli.Bucket(bucketName)
			if err != nil {
				return false, err
			}
			objectName := outputArtifact.OSS.Key
			err = bucket.PutObjectFromFile(objectName, path)
			if err != nil {
				return false, err
			}
			return true, nil
		})
	return err
}
