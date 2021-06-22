package oss

import (
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/argoproj/pkg/file"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// ArtifactDriver is a driver for OSS
type ArtifactDriver struct {
	Endpoint      string
	AccessKey     string
	SecretKey     string
	SecurityToken string
}

var (
	_            common.ArtifactDriver = &ArtifactDriver{}
	defaultRetry                       = wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1}

	// OSS error code reference: https://error-center.alibabacloud.com/status/product/Oss
	ossTransientErrorCodes = []string{"RequestTimeout", "QuotaExceeded.Refresh", "Default", "ServiceUnavailable", "Throttling", "RequestTimeTooSkewed", "SocketException", "SocketTimeout", "ServiceBusy", "DomainNetWorkVisitedException", "ConnectionTimeout", "CachedTimeTooLarge"}
)

func (ossDriver *ArtifactDriver) newOSSClient() (*oss.Client, error) {
	var options []oss.ClientOption
	if token := ossDriver.SecurityToken; token != "" {
		options = append(options, oss.SecurityToken(token))
	}
	client, err := oss.New(ossDriver.Endpoint, ossDriver.AccessKey, ossDriver.SecretKey, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create new OSS client: %w", err)
	}
	return client, err
}

// Downloads artifacts from OSS compliant storage, e.g., downloading an artifact into local path
func (ossDriver *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			log.Infof("OSS Load path: %s, key: %s", path, inputArtifact.OSS.Key)
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			bucketName := inputArtifact.OSS.Bucket
			bucket, err := osscli.Bucket(bucketName)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			objectName := inputArtifact.OSS.Key
			origErr := bucket.GetObjectToFile(objectName, path)
			if origErr == nil {
				return true, nil
			}
			if ossErr, ok := origErr.(oss.ServiceError); ok {
				if ossErr.Code != "NoSuchKey" {
					return !isTransientOSSErr(err), fmt.Errorf("failed to get file: %w", origErr)
				}
			}
			// If we get here, the error was a NoSuchKey. The key might be a directory.
			// There is only one method in OSS for downloading objects that does not differentiate between a file
			// and a directory so we append the a trailing slash here to differentiate that prior to downloading.
			err = bucket.GetObjectToFile(objectName+"/", path)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			return true, nil
		})
	return err
}

// Saves an artifact to OSS compliant storage, e.g., uploading a local file to OSS bucket
func (ossDriver *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			log.Infof("OSS Save path: %s, key: %s", path, outputArtifact.OSS.Key)
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			bucketName := outputArtifact.OSS.Bucket
			if outputArtifact.OSS.CreateBucketIfNotPresent {
				exists, err := osscli.IsBucketExist(bucketName)
				if err != nil {
					return !isTransientOSSErr(err), fmt.Errorf("failed to check if bucket %s exists: %w", bucketName, err)
				}
				if !exists {
					err = osscli.CreateBucket(bucketName)
					if err != nil {
						return !isTransientOSSErr(err), fmt.Errorf("failed to automatically create bucket %s when it's not present: %w", bucketName, err)
					}
				}
			}
			bucket, err := osscli.Bucket(bucketName)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			objectName := outputArtifact.OSS.Key
			if outputArtifact.OSS.LifecycleRule != nil {
				err = setBucketLifecycleRule(osscli, outputArtifact.OSS)
				return !isTransientOSSErr(err), err
			}
			isDir, err := file.IsDirectory(path)
			if err != nil {
				return false, fmt.Errorf("failed to test if %s is a directory: %w", path, err)
			}
			// There is only one method in OSS for uploading objects that does not differentiate between a file and a directory
			// so we append the a trailing slash here to differentiate that prior to uploading.
			if isDir && !strings.HasSuffix(objectName, "/") {
				objectName += "/"
			}
			err = bucket.PutObjectFromFile(objectName, path)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			return true, nil
		})
	return err
}

func (ossDriver *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	var files []string
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			bucket, err := osscli.Bucket(artifact.OSS.Bucket)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			results, err := bucket.ListObjectsV2(oss.Prefix(artifact.OSS.Key))
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			for _, object := range results.Objects {
				files = append(files, object.Key)
			}
			return true, nil
		})
	return files, err
}

func setBucketLifecycleRule(client *oss.Client, ossArtifact *wfv1.OSSArtifact) error {
	if ossArtifact.LifecycleRule.MarkInfrequentAccessAfterDays == 0 && ossArtifact.LifecycleRule.MarkDeletionAfterDays == 0 {
		return nil
	}
	var markInfrequentAccessAfterDays int
	var markDeletionAfterDays int
	if ossArtifact.LifecycleRule.MarkInfrequentAccessAfterDays != 0 {
		markInfrequentAccessAfterDays = int(ossArtifact.LifecycleRule.MarkInfrequentAccessAfterDays)
	}
	if ossArtifact.LifecycleRule.MarkDeletionAfterDays != 0 {
		markDeletionAfterDays = int(ossArtifact.LifecycleRule.MarkDeletionAfterDays)
	}
	if markInfrequentAccessAfterDays > markDeletionAfterDays {
		return fmt.Errorf("markInfrequentAccessAfterDays cannot be large than markDeletionAfterDays")
	}

	// Set expiration rule.
	expirationRule := oss.BuildLifecycleRuleByDays("expiration-rule", ossArtifact.Key, true, markInfrequentAccessAfterDays)
	// Automatically delete the expired delete tag so we don't have to manage it ourselves.
	expiration := oss.LifecycleExpiration{
		ExpiredObjectDeleteMarker: pointer.BoolPtr(true),
	}
	// Convert to Infrequent Access (IA) storage type for objects that are expired after a period of time.
	versionTransition := oss.LifecycleVersionTransition{
		NoncurrentDays: markInfrequentAccessAfterDays,
		StorageClass:   oss.StorageIA,
	}
	// Mark deletion after a period of time.
	versionExpiration := oss.LifecycleVersionExpiration{
		NoncurrentDays: markDeletionAfterDays,
	}
	versionTransitionRule := oss.LifecycleRule{
		ID:                    "version-transition-rule",
		Prefix:                ossArtifact.Key,
		Status:                string(oss.VersionEnabled),
		Expiration:            &expiration,
		NonVersionExpiration:  &versionExpiration,
		NonVersionTransitions: []oss.LifecycleVersionTransition{versionTransition},
	}

	// Set lifecycle rules to the bucket.
	rules := []oss.LifecycleRule{expirationRule, versionTransitionRule}
	err := client.SetBucketLifecycle(ossArtifact.Bucket, rules)
	return err
}

func isTransientOSSErr(err error) bool {
	if err == nil {
		return false
	}
	if ossErr, ok := err.(oss.ServiceError); ok {
		for _, transientErrCode := range ossTransientErrorCodes {
			if ossErr.Code == transientErrCode {
				return true
			}
		}
	}
	return false
}
