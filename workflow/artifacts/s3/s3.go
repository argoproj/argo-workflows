package s3

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/argoproj/pkg/file"
	argos3 "github.com/argoproj/pkg/s3"
	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/util/retry"

	argoerrs "github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	artifactscommon "github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	executorretry "github.com/argoproj/argo-workflows/v3/workflow/executor/retry"
)

// ArtifactDriver is a driver for AWS S3
type ArtifactDriver struct {
	Endpoint              string
	Region                string
	Secure                bool
	TrustedCA             string
	AccessKey             string
	SecretKey             string
	SessionToken          string
	RoleARN               string
	UseSDKCreds           bool
	Context               context.Context
	KmsKeyId              string
	KmsEncryptionContext  string
	EnableEncryption      bool
	ServerSideCustomerKey string
}

var _ artifactscommon.ArtifactDriver = &ArtifactDriver{}

// newS3Client instantiates a new S3 client object.
func (s3Driver *ArtifactDriver) newS3Client(ctx context.Context) (argos3.S3Client, error) {
	opts := argos3.S3ClientOpts{
		Endpoint:     s3Driver.Endpoint,
		Region:       s3Driver.Region,
		Secure:       s3Driver.Secure,
		AccessKey:    s3Driver.AccessKey,
		SecretKey:    s3Driver.SecretKey,
		SessionToken: s3Driver.SessionToken,
		RoleARN:      s3Driver.RoleARN,
		Trace:        os.Getenv(common.EnvVarArgoTrace) == "1",
		UseSDKCreds:  s3Driver.UseSDKCreds,
		EncryptOpts: argos3.EncryptOpts{
			KmsKeyId:              s3Driver.KmsKeyId,
			KmsEncryptionContext:  s3Driver.KmsEncryptionContext,
			Enabled:               s3Driver.EnableEncryption,
			ServerSideCustomerKey: s3Driver.ServerSideCustomerKey,
		},
	}

	if tr, err := argos3.GetDefaultTransport(opts); err == nil {
		if s3Driver.Secure && s3Driver.TrustedCA != "" {
			// Trust only the provided root CA
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM([]byte(s3Driver.TrustedCA))
			tr.TLSClientConfig.RootCAs = pool
		}
		opts.Transport = tr
	}

	return argos3.NewS3Client(ctx, opts)
}

// Load downloads artifacts from S3 compliant storage
func (s3Driver *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := waitutil.Backoff(executorretry.ExecutorRetry,
		func() (bool, error) {
			log.Infof("S3 Load path: %s, key: %s", path, inputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return !isTransientS3Err(err), fmt.Errorf("failed to create new S3 client: %v", err)
			}
			return loadS3Artifact(s3cli, inputArtifact, path)
		})

	return err
}

// loadS3Artifact downloads artifacts from an S3 compliant storage
// returns true if the download is completed or can't be retried (non-transient error)
// returns false if it can be retried (transient error)
func loadS3Artifact(s3cli argos3.S3Client, inputArtifact *wfv1.Artifact, path string) (bool, error) {
	origErr := s3cli.GetFile(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path)
	if origErr == nil {
		return true, nil
	}

	if strings.Contains(origErr.Error(), "fileName is a directory.") {
		// Handle directory case by checking if it's a valid directory
		isDir, err := s3cli.IsDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
		if err != nil {
			return !isTransientS3Err(err), fmt.Errorf("failed to test if %s is a directory: %v", inputArtifact.S3.Key, err)
		}
		if isDir {
			// Proceed to get the directory contents if it's actually a directory
			if err := s3cli.GetDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path); err != nil {
				return !isTransientS3Err(err), fmt.Errorf("failed to get directory: %v", err)
			}
			return true, nil
		}
	}

	if !argos3.IsS3ErrCode(origErr, "NoSuchKey") {
		return !isTransientS3Err(origErr), fmt.Errorf("failed to get file: %v", origErr)
	}
	// If we get here, the error was a NoSuchKey. The key might be an s3 "directory"
	isDir, err := s3cli.IsDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if err != nil {
		return !isTransientS3Err(err), fmt.Errorf("failed to test if %s is a directory: %v", inputArtifact.S3.Key, err)
	}
	if !isDir {
		// It's neither a file, nor a directory. Return the original NoSuchKey error
		return true, argoerrs.New(argoerrs.CodeNotFound, origErr.Error())
	}

	if err = s3cli.GetDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key, path); err != nil {
		return !isTransientS3Err(err), fmt.Errorf("failed to get directory: %v", err)
	}
	return true, nil
}

// OpenStream opens a stream reader for an artifact from S3 compliant storage
func (s3Driver *ArtifactDriver) OpenStream(inputArtifact *wfv1.Artifact) (io.ReadCloser, error) {
	log.Infof("S3 OpenStream: key: %s", inputArtifact.S3.Key)
	s3cli, err := s3Driver.newS3Client(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to create new S3 client: %v", err)
	}

	return streamS3Artifact(s3cli, inputArtifact)

}

func streamS3Artifact(s3cli argos3.S3Client, inputArtifact *wfv1.Artifact) (io.ReadCloser, error) {
	stream, origErr := s3cli.OpenFile(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if origErr == nil {
		return stream, nil
	}
	if !argos3.IsS3ErrCode(origErr, "NoSuchKey") {
		return nil, fmt.Errorf("failed to get file: %v", origErr)
	}
	// If we get here, the error was a NoSuchKey. The key might be an s3 "directory"
	isDir, err := s3cli.IsDirectory(inputArtifact.S3.Bucket, inputArtifact.S3.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to test if %s is a directory: %v", inputArtifact.S3.Key, err)
	}
	if !isDir {
		// It's neither a file, nor a directory. Return the original NoSuchKey error
		return nil, argoerrs.New(argoerrs.CodeNotFound, origErr.Error())
	}
	// directory case:
	// todo: make a .tgz file which can be streamed to user
	return nil, argoerrs.New(argoerrs.CodeNotImplemented, "Directory Stream capability currently unimplemented for S3")
}

// Save saves an artifact to S3 compliant storage
func (s3Driver *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := waitutil.Backoff(executorretry.ExecutorRetry,
		func() (bool, error) {
			log.Infof("S3 Save path: %s, key: %s", path, outputArtifact.S3.Key)
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return !isTransientS3Err(err), fmt.Errorf("failed to create new S3 client: %v", err)
			}
			return saveS3Artifact(s3cli, path, outputArtifact)
		})
	return err
}

// Delete deletes an artifact from an S3 compliant storage
func (s3Driver *ArtifactDriver) Delete(artifact *wfv1.Artifact) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := retry.OnError(retry.DefaultBackoff, isTransientS3Err, func() error {
		log.Infof("S3 Delete artifact: key: %s", artifact.S3.Key)
		s3cli, err := s3Driver.newS3Client(ctx)
		if err != nil {
			return err
		}

		// check suffix instead of s3cli.IsDirectory as it requires another request for file delete (most scenarios)
		if !strings.HasSuffix(artifact.S3.Key, "/") {
			return s3cli.Delete(artifact.S3.Bucket, artifact.S3.Key)
		}

		keys, err := s3cli.ListDirectory(artifact.S3.Bucket, artifact.S3.Key)
		if err != nil {
			return fmt.Errorf("unable to list files in %s: %s", artifact.S3.Key, err)
		}
		for _, objKey := range keys {
			err = s3cli.Delete(artifact.S3.Bucket, objKey)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

// saveS3Artifact uploads artifacts to an S3 compliant storage
// returns true if the upload is completed or can't be retried (non-transient error)
// returns false if it can be retried (transient error)
func saveS3Artifact(s3cli argos3.S3Client, path string, outputArtifact *wfv1.Artifact) (bool, error) {
	isDir, err := file.IsDirectory(path)
	if err != nil {
		return true, fmt.Errorf("failed to test if %s is a directory: %v", path, err)
	}

	createBucketIfNotPresent := outputArtifact.S3.CreateBucketIfNotPresent
	if createBucketIfNotPresent != nil {
		log.WithField("bucket", outputArtifact.S3.Bucket).Info("creating bucket")
		err := s3cli.MakeBucket(outputArtifact.S3.Bucket, minio.MakeBucketOptions{
			Region:        outputArtifact.S3.Region,
			ObjectLocking: outputArtifact.S3.CreateBucketIfNotPresent.ObjectLocking,
		})
		alreadyExists := bucketAlreadyExistsErr(err)
		log.WithField("bucket", outputArtifact.S3.Bucket).
			WithField("alreadyExists", alreadyExists).
			WithError(err).
			Info("create bucket failed")
		if err != nil && !alreadyExists {
			return !isTransientS3Err(err), fmt.Errorf("failed to create bucket %s: %v", outputArtifact.S3.Bucket, err)
		}
	}

	if isDir {
		if err = s3cli.PutDirectory(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path); err != nil {
			return !isTransientS3Err(err), fmt.Errorf("failed to put directory: %v", err)
		}
	} else {
		if err = s3cli.PutFile(outputArtifact.S3.Bucket, outputArtifact.S3.Key, path); err != nil {
			return !isTransientS3Err(err), fmt.Errorf("failed to put file: %v", err)
		}
	}
	return true, nil
}

func bucketAlreadyExistsErr(err error) bool {
	resp := &minio.ErrorResponse{}
	// https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html
	alreadyExistsCodes := map[string]bool{"BucketAlreadyExists": true, "BucketAlreadyOwnedByYou": true}
	return errors.As(err, resp) && alreadyExistsCodes[resp.Code]
}

// ListObjects returns the files inside the directory represented by the Artifact
func (s3Driver *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var files []string
	var done bool
	err := waitutil.Backoff(executorretry.ExecutorRetry,
		func() (bool, error) {
			s3cli, err := s3Driver.newS3Client(ctx)
			if err != nil {
				return !isTransientS3Err(err), fmt.Errorf("failed to create new S3 client: %v", err)
			}
			done, files, err = listObjects(s3cli, artifact)
			return done, err
		})

	return files, err
}

// listObjects returns the files inside the directory represented by the Artifact
// returns true if success or can't be retried (non-transient error)
// returns false if it can be retried (transient error)
func listObjects(s3cli argos3.S3Client, artifact *wfv1.Artifact) (bool, []string, error) {
	var files []string
	files, err := s3cli.ListDirectory(artifact.S3.Bucket, artifact.S3.Key)
	if err != nil {
		return !isTransientS3Err(err), files, fmt.Errorf("failed to list directory: %v", err)
	}
	log.Debugf("successfully listing S3 directory associated with bucket: %s and key %s: %v", artifact.S3.Bucket, artifact.S3.Key, files)

	if len(files) == 0 {
		directoryExists, err := s3cli.KeyExists(artifact.S3.Bucket, artifact.S3.Key)
		if err != nil {
			return !isTransientS3Err(err), files, fmt.Errorf("failed to check if key %s exists from bucket %s: %v", artifact.S3.Key, artifact.S3.Bucket, err)
		}
		if !directoryExists {
			return true, files, argoerrs.New(argoerrs.CodeNotFound, fmt.Sprintf("no key found of name %s", artifact.S3.Key))
		}
	}
	return true, files, nil
}

func (s3Driver *ArtifactDriver) IsDirectory(artifact *wfv1.Artifact) (bool, error) {
	s3cli, err := s3Driver.newS3Client(context.TODO())
	if err != nil {
		return false, err
	}
	return s3cli.IsDirectory(artifact.S3.Bucket, artifact.S3.Key)
}
