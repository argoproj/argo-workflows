package oss

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/argoproj/pkg/file"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/credentials-go/credentials"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errutil "github.com/argoproj/argo-workflows/v3/util/errors"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	wfcommon "github.com/argoproj/argo-workflows/v3/workflow/common"
)

// ArtifactDriver is a driver for OSS
type ArtifactDriver struct {
	Endpoint      string
	AccessKey     string
	SecretKey     string
	SecurityToken string
	UseSDKCreds   bool
}

var (
	_            common.ArtifactDriver = &ArtifactDriver{}
	defaultRetry                       = wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1}

	// OSS error code reference: https://error-center.alibabacloud.com/status/product/Oss
	ossTransientErrorCodes = []string{"RequestTimeout", "QuotaExceeded.Refresh", "Default", "ServiceUnavailable", "Throttling", "RequestTimeTooSkewed", "SocketException", "SocketTimeout", "ServiceBusy", "DomainNetWorkVisitedException", "ConnectionTimeout", "CachedTimeTooLarge"}
	bucketLogFilePrefix    = "bucket-log-"
)

type ossCredentials struct {
	teaCred credentials.Credential
}

func (cred *ossCredentials) GetAccessKeyID() string {
	value, err := cred.teaCred.GetAccessKeyId()
	if err != nil {
		log.Infof("get access key id failed: %+v", err)
		return ""
	}
	return tea.StringValue(value)
}

func (cred *ossCredentials) GetAccessKeySecret() string {
	value, err := cred.teaCred.GetAccessKeySecret()
	if err != nil {
		log.Infof("get access key secret failed: %+v", err)
		return ""
	}
	return tea.StringValue(value)
}

func (cred *ossCredentials) GetSecurityToken() string {
	value, err := cred.teaCred.GetSecurityToken()
	if err != nil {
		log.Infof("get access security token failed: %+v", err)
		return ""
	}
	return tea.StringValue(value)
}

type ossCredentialsProvider struct {
	cred credentials.Credential
}

func (p *ossCredentialsProvider) GetCredentials() oss.Credentials {
	return &ossCredentials{teaCred: p.cred}
}

func (ossDriver *ArtifactDriver) newOSSClient() (*oss.Client, error) {
	var options []oss.ClientOption
	if token := ossDriver.SecurityToken; token != "" {
		options = append(options, oss.SecurityToken(token))
	}
	if ossDriver.UseSDKCreds {
		// using default provider chains in sdk to get credential
		log.Infof("Using default sdk provider chains for OSS driver")
		// need install ack-pod-identity-webhook in your cluster when using oidc provider for OSS drirver
		// the mutating webhook will help to inject the required OIDC env variables and toke volume mount configuration
		// please refer to https://www.alibabacloud.com/help/en/ack/product-overview/ack-pod-identity-webhook
		cred, err := credentials.NewCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create new OSS client: %w", err)
		}
		provider := &ossCredentialsProvider{cred: cred}
		return oss.New(ossDriver.Endpoint, "", "", oss.SetCredentialsProvider(provider))
	}
	log.Infof("Using AK provider")
	client, err := oss.New(ossDriver.Endpoint, ossDriver.AccessKey, ossDriver.SecretKey, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create new OSS client: %w", err)
	}
	return client, err
}

// Load downloads artifacts from OSS compliant storage, e.g., downloading an artifact into local path
func (ossDriver *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			log.Infof("OSS Load path: %s, key: %s", path, inputArtifact.OSS.Key)
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			bucketName := inputArtifact.OSS.Bucket
			err = setBucketLogging(osscli, bucketName)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			bucket, err := osscli.Bucket(bucketName)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			objectName := inputArtifact.OSS.Key
			dirPath := filepath.Dir(path)
			err = os.MkdirAll(dirPath, 0o700)
			if err != nil {
				return false, fmt.Errorf("mkdir %s error: %w", dirPath, err)
			}
			origErr := bucket.GetObjectToFile(objectName, path)
			if origErr == nil {
				return true, nil
			}
			if !IsOssErrCode(origErr, "NoSuchKey") {
				return !isTransientOSSErr(origErr), fmt.Errorf("failed to get file: %w", origErr)
			}
			// If we get here, the error was a NoSuchKey. The key might be a oss "directory"
			isDir, err := IsOssDirectory(bucket, objectName)
			if err != nil {
				return !isTransientOSSErr(err), fmt.Errorf("failed to test if %s/%s is a directory: %w", bucketName, objectName, err)
			}
			if !isDir {
				// It's neither a file, nor a directory. Return the original NoSuchKey error
				return false, origErr
			}

			if err = GetOssDirectory(bucket, objectName, path); err != nil {
				return !isTransientOSSErr(err), fmt.Errorf("failed get directory: %v", err)
			}
			return true, nil
		})
	return err
}

func (ossDriver *ArtifactDriver) OpenStream(a *wfv1.Artifact) (io.ReadCloser, error) {
	// todo: this is a temporary implementation which loads file to disk first
	return common.LoadToStream(a, ossDriver)
}

// Save stores an artifact to OSS compliant storage, e.g., uploading a local file to OSS bucket
func (ossDriver *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			log.Infof("OSS Save path: %s, key: %s", path, outputArtifact.OSS.Key)
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			isDir, err := file.IsDirectory(path)
			if err != nil {
				log.Warnf("Failed to test if %s is a directory: %v", path, err)
				return false, nil
			}
			bucketName := outputArtifact.OSS.Bucket
			err = setBucketLogging(osscli, bucketName)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
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
			if isDir {
				if err = putDirectory(bucket, objectName, path); err != nil {
					log.Warnf("failed to put directory: %v", err)
					return !isTransientOSSErr(err), err
				}
			} else {
				if err = putFile(bucket, objectName, path); err != nil {
					log.Warnf("failed to put file: %v", err)
					return !isTransientOSSErr(err), err
				}
			}

			return true, nil
		})
	return err
}

// Delete is unsupported for the oss artifacts
func (ossDriver *ArtifactDriver) Delete(s *wfv1.Artifact) error {
	return common.ErrDeleteNotSupported
}

func (ossDriver *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	var files []string
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			osscli, err := ossDriver.newOSSClient()
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			bucketName := artifact.OSS.Bucket
			err = setBucketLogging(osscli, bucketName)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			bucket, err := osscli.Bucket(bucketName)
			if err != nil {
				return !isTransientOSSErr(err), err
			}
			pre := oss.Prefix(artifact.OSS.Key)
			continueToken := ""
			for {
				results, err := bucket.ListObjectsV2(pre, oss.ContinuationToken(continueToken))
				if err != nil {
					return !isTransientOSSErr(err), err
				}
				// Add to files. By default, 100 records are returned at a time. https://help.aliyun.com/zh/oss/user-guide/list-objects-18
				for _, object := range results.Objects {
					files = append(files, object.Key)
				}
				if results.IsTruncated {
					continueToken = results.NextContinuationToken
					pre = oss.Prefix(results.Prefix)
				} else {
					break
				}
			}
			return true, nil
		})
	return files, err
}

func setBucketLogging(client *oss.Client, bucketName string) error {
	if os.Getenv(wfcommon.EnvVarArgoTrace) == "1" {
		err := client.SetBucketLogging(bucketName, bucketName, bucketLogFilePrefix, true)
		if err != nil {
			return fmt.Errorf("failed to configure bucket logging: %w", err)
		}
	}
	return nil
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
		ExpiredObjectDeleteMarker: pointer.Bool(true),
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
	if errutil.IsTransientErr(err) {
		return true
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

func putFile(bucket *oss.Bucket, objectName, path string) error {
	log.Debugf("putFile from %s to %s", path, objectName)
	return bucket.PutObjectFromFile(objectName, path)
}

func putDirectory(bucket *oss.Bucket, objectName, dir string) error {
	return filepath.Walk(dir, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.InternalWrapError(err)
		}
		// build the name to be used in OSS
		nameInDir, err := filepath.Rel(dir, fpath)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		fObjectName := filepath.Join(objectName, nameInDir)
		// create an OSS dir explicitly for every local dir, , including empty dirs.
		if info.Mode().IsDir() {
			// create OSS dir
			if !strings.HasSuffix(fObjectName, "/") {
				fObjectName += "/"
			}
			err = bucket.PutObject(fObjectName, nil)
			if err != nil {
				return err
			}
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		err = putFile(bucket, fObjectName, fpath)
		if err != nil {
			return err
		}
		return nil
	})
}

// IsOssErrCode tests if an err is an oss.ServiceError with the specified code
func IsOssErrCode(err error, code string) bool {
	if serr, ok := err.(oss.ServiceError); ok {
		if serr.Code == code {
			return true
		}
	}
	return false
}

// IsOssDirectory tests if the key is acting like a OSS directory. This just means it has at least one
// object which is prefixed with the given key
func IsOssDirectory(bucket *oss.Bucket, objectName string) (bool, error) {
	if objectName == "" {
		return true, nil
	}
	if !strings.HasSuffix(objectName, "/") {
		objectName += "/"
	}
	rst, err := bucket.ListObjects(oss.Prefix(objectName), oss.MaxKeys(1))
	if err != nil {
		return false, err
	}
	if len(rst.CommonPrefixes)+len(rst.Objects) > 0 {
		return true, nil
	}
	return false, nil
}

// GetOssDirectory download an OSS "directory" to local path
func GetOssDirectory(bucket *oss.Bucket, objectName, path string) error {
	files, err := ListOssDirectory(bucket, objectName)
	if err != nil {
		return err
	}
	for _, f := range files {
		innerName, err := filepath.Rel(objectName, f)
		if err != nil {
			return fmt.Errorf("get Rel path from %s to %s error: %w", f, objectName, err)
		}
		fpath := filepath.Join(path, innerName)
		if strings.HasSuffix(f, "/") {
			err = os.MkdirAll(fpath, 0o700)
			if err != nil {
				return fmt.Errorf("mkdir %s error: %w", fpath, err)
			}
			continue
		}
		dirPath := filepath.Dir(fpath)
		err = os.MkdirAll(dirPath, 0o700)
		if err != nil {
			return fmt.Errorf("mkdir %s error: %w", dirPath, err)
		}

		err = bucket.GetObjectToFile(f, fpath)
		if err != nil {
			log.Warnf("failed to load object %s to %s error: %v", f, fpath, err)
			return err
		}
	}
	return nil
}

// ListOssDirectory lists all the files which are the descendants of the specified objectKey, if a file has suffix '/', then it is an OSS directory
func ListOssDirectory(bucket *oss.Bucket, objectKey string) (files []string, err error) {
	if objectKey != "" {
		if !strings.HasSuffix(objectKey, "/") {
			objectKey += "/"
		}
	}

	pre := oss.Prefix(objectKey)
	marker := oss.Marker("")
	for {
		lor, err := bucket.ListObjects(marker, pre)
		if err != nil {
			log.Warnf("oss list object(%s) error: %v", objectKey, err)
			return files, err
		}
		for _, obj := range lor.Objects {
			files = append(files, obj.Key)
		}

		marker = oss.Marker(lor.NextMarker)
		if !lor.IsTruncated {
			break
		}
	}
	return files, nil
}

// IsDirectory tests if the key is acting like a OSS directory
func (ossDriver *ArtifactDriver) IsDirectory(artifact *wfv1.Artifact) (bool, error) {
	osscli, err := ossDriver.newOSSClient()
	if err != nil {
		return !isTransientOSSErr(err), err
	}
	bucketName := artifact.OSS.Bucket
	bucket, err := osscli.Bucket(bucketName)
	if err != nil {
		return !isTransientOSSErr(err), err
	}
	objectName := artifact.OSS.Key
	isDir, err := IsOssDirectory(bucket, objectName)
	if err != nil {
		return !isTransientOSSErr(err), fmt.Errorf("failed to test if %s/%s is a directory: %w", bucketName, objectName, err)
	}
	return isDir, nil
}
