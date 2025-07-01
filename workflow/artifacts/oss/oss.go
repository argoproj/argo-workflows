package oss

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/credentials-go/credentials"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/file"
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
	ossTransientErrorCodes = []string{"RequestTimeout", "QuotaExceeded.Refresh", "Default", "ServiceUnavailable", "Throttling", "RequestTimeTooSkewed", "SocketException", "SocketTimeout", "ServiceBusy", "DomainNetWorkVisitedException", "ConnectionTimeout", "CachedTimeTooLarge", "InternalError"}
	bucketLogFilePrefix    = "bucket-log-"
	maxObjectSize          = int64(5 * 1024 * 1024 * 1024)
)

type ossCredentials struct {
	teaCred credentials.Credential
}

func (cred *ossCredentials) GetAccessKeyID() string {
	credential, err := cred.teaCred.GetCredential()
	if err != nil {
		log.Infof("get access key id failed: %+v", err)
		return ""
	}
	return tea.StringValue(credential.AccessKeyId)
}

func (cred *ossCredentials) GetAccessKeySecret() string {
	credential, err := cred.teaCred.GetCredential()
	if err != nil {
		log.Infof("get access key secret failed: %+v", err)
		return ""
	}
	return tea.StringValue(credential.AccessKeySecret)
}

func (cred *ossCredentials) GetSecurityToken() string {
	credential, err := cred.teaCred.GetCredential()
	if err != nil {
		log.Infof("get access security token failed: %+v", err)
		return ""
	}
	return tea.StringValue(credential.SecurityToken)
}

type ossCredentialsProvider struct {
	cred credentials.Credential
}

func (p *ossCredentialsProvider) GetCredentials() oss.Credentials {
	return &ossCredentials{teaCred: p.cred}
}

func (ossDriver *ArtifactDriver) newOSSClient() (*oss.Client, error) {
	var options []oss.ClientOption

	// for oss driver, the proxy cannot be configured through environment variables
	// ref: https://help.aliyun.com/zh/cli/use-an-http-proxy-server#section-5yf-ejl-jwf
	if proxy, ok := os.LookupEnv("https_proxy"); ok {
		options = append(options, oss.Proxy(proxy))
	}

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

// OpenStream opens a stream reader for an artifact from OSS compliant storage
func (ossDriver *ArtifactDriver) OpenStream(inputArtifact *wfv1.Artifact) (io.ReadCloser, error) {
	var stream io.ReadCloser
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			log.Infof("OSS OpenStream, key: %s", inputArtifact.OSS.Key)
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
			s, origErr := bucket.GetObject(inputArtifact.OSS.Key)
			if origErr == nil {
				stream = s
				return true, nil
			}
			if !IsOssErrCode(err, "NoSuchKey") {
				return !isTransientOSSErr(origErr), fmt.Errorf("failed to get file: %w", origErr)
			}
			isDir, err := IsOssDirectory(bucket, inputArtifact.OSS.Key)
			if err != nil {
				return !isTransientOSSErr(err), fmt.Errorf("failed to test if %s/%s is a directory: %w", bucketName, inputArtifact.OSS.Key, err)
			}
			if !isDir {
				return false, origErr
			}
			// directory case:
			// todo: make a .tgz file which can be streamed to user
			return false, errors.New(errors.CodeNotImplemented, "Directory Stream capability currently unimplemented for OSS")
		})
	return stream, err
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
				if err != nil {
					return !isTransientOSSErr(err), err
				}
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

// Delete deletes an artifact from an OSS compliant storage
func (ossDriver *ArtifactDriver) Delete(artifact *wfv1.Artifact) error {
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			log.Infof("OSS Delete artifact: key: %s", artifact.OSS.Key)
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
			objectName := artifact.OSS.Key
			err = bucket.DeleteObject(objectName)
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

	// Delete the current version objects after a period of time.
	// If BucketVersioning is enbaled, the objects will turn to non-current version.
	expiration := oss.LifecycleExpiration{
		Days: markDeletionAfterDays,
	}
	// Convert to Infrequent Access (IA) storage type for objects that are expired after a period of time.
	transition := oss.LifecycleTransition{
		Days:         markInfrequentAccessAfterDays,
		StorageClass: oss.StorageIA,
	}
	// Delete the aborted uploaded parts after a period of time.
	abortMultipartUpload := oss.LifecycleAbortMultipartUpload{
		Days: markDeletionAfterDays,
	}

	keySha := fmt.Sprintf("%x", sha256.Sum256([]byte(ossArtifact.Key)))
	rule := oss.LifecycleRule{
		ID:                   keySha,
		Prefix:               ossArtifact.Key,
		Status:               string(oss.VersionEnabled),
		Expiration:           &expiration,
		Transitions:          []oss.LifecycleTransition{transition},
		AbortMultipartUpload: &abortMultipartUpload,
	}

	// Set lifecycle rules to the bucket.
	err := client.SetBucketLifecycle(ossArtifact.Bucket, []oss.LifecycleRule{rule})
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

// OSS simple upload code reference: https://www.alibabacloud.com/help/en/oss/user-guide/simple-upload?spm=a2c63.p38356.0.0.2c072398fh5k3W#section-ym8-svm-rmu
func simpleUpload(bucket *oss.Bucket, objectName, path string) error {
	log.Info("OSS Simple Uploading")
	return bucket.PutObjectFromFile(objectName, path)
}

// OSS multipart upload code reference: https://www.alibabacloud.com/help/en/oss/user-guide/multipart-upload?spm=a2c63.p38356.0.0.4ebe423fzsaPiN#section-trz-mpy-tes
func multipartUpload(bucket *oss.Bucket, objectName, path string, objectSize int64) error {
	log.Info("OSS Multipart Uploading")
	// Calculate the number of chunks
	chunkNum := int(math.Ceil(float64(objectSize)/float64(maxObjectSize))) + 1
	chunks, err := oss.SplitFileByPartNum(path, chunkNum)
	if err != nil {
		return err
	}
	fd, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer fd.Close()
	// Initialize a multipart upload event.
	imur, err := bucket.InitiateMultipartUpload(objectName)
	if err != nil {
		return err
	}
	// Upload the chunks.
	var parts []oss.UploadPart
	for _, chunk := range chunks {
		_, err := fd.Seek(chunk.Offset, io.SeekStart)
		if err != nil {
			return err
		}
		// Call the UploadPart method to upload each chunck.
		part, err := bucket.UploadPart(imur, fd, chunk.Size, chunk.Number)
		if err != nil {
			log.Warnf("Upload part error: %v", err)
			return err
		}
		log.Infof("Upload part number: %v, ETag: %v", part.PartNumber, part.ETag)
		parts = append(parts, part)
	}
	_, err = bucket.CompleteMultipartUpload(imur, parts)
	if err != nil {
		log.Warnf("Complete multipart upload error: %v", err)
		return err
	}
	return nil
}

func putFile(bucket *oss.Bucket, objectName, path string) error {
	log.Debugf("putFile from %s to %s", path, objectName)
	fStat, err := os.Stat(path)
	if err != nil {
		return err
	}
	// Determine upload method based on file size.
	if fStat.Size() <= maxObjectSize {
		return simpleUpload(bucket, objectName, path)
	}
	return multipartUpload(bucket, objectName, path, fStat.Size())
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
