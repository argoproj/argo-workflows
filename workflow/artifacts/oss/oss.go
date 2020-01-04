package oss

import (

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argooss "github.com/argoproj/pkg/oss"
	"github.com/argoproj/pkg/file"
	"time"
)

// OSSArtifactDriver is a driver for OSS
type OSSArtifactDriver struct {
	Endpoint  string
	AccessKey string
	SecretKey string
}


// newMinioClient instantiates a new minio client object.
func (ossDriver *OSSArtifactDriver) newOSSClient() (argooss.OSSClient, error) {
	opts := argooss.OSSClientOpts{
		Endpoint:  ossDriver.Endpoint,
		AccessKey: ossDriver.AccessKey,
		SecretKey: ossDriver.SecretKey,
	}
	return argooss.NewOSSClient(opts)
}

// Load downloads artifacts from OSS compliant storage
func (ossDriver *OSSArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("OSS Load path: %s, key: %s", path, inputArtifactOSSOSS.Key)
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				log.Warnf("Failed to create new OSS client: %v", err)
				return false, nil
			}
			origErr := osscli.GetFile(inputArtifact.oss.Bucket, inputArtifact.oss.Key, path)
			if origErr == nil {
				return true, nil
			}
			if !argooss.IsOSSErrCode(origErr, "NoSuchKey") {
				log.Warnf("Failed get file: %v", origErr)
				return false, nil
			}
			// If we get here, the error was a NoSuchKey. The key might be a OSS "directory"
			isDir, err := osscli.IsDirectory(inputArtifact.oss.Bucket, inputArtifact.oss.Key)
			if err != nil {
				log.Warnf("Failed to test if %s is a directory: %v", inputArtifact.oss.Bucket, err)
				return false, nil
			}
			if !isDir {
				// It's neither a file, nor a directory. Return the original NoSuchKey error
				return false, origErr
			}

			if err = osscli.GetDirectory(inputArtifact.oss.Bucket, inputArtifact.oss.Key, path); err != nil {
				log.Warnf("Failed get directory: %v", err)
				return false, nil
			}
			return true, nil
		})

	return err
}

// Save saves an artifact to OSS compliant storage
func (ossDriver *OSSArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {

	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("S3 Save path: %s, key: %s", path, outputArtifact.oss.Key)
			osscli, err := ossDriver.newOSSClient()
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
				if err = osscli.PutDirectory(outputArtifact.oss.Bucket, outputArtifact.oss.Key, path); err != nil {
					log.Warnf("Failed to put directory: %v", err)
					return false, nil
				}
			} else {
				if err = osscli.PutFile(outputArtifact.oss.Bucket, outputArtifact.oss.Key, path); err != nil {
					log.Warnf("Failed to put file: %v", err)
					return false, nil
				}
			}
			return true, nil
		})
	return err

}