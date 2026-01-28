package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	argoerrors "github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/file"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// ArtifactDriver is a driver for GCS
type ArtifactDriver struct {
	ServiceAccountKey string
}

var (
	_            common.ArtifactDriver = &ArtifactDriver{}
	defaultRetry                       = wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1, Cap: time.Minute * 10}
)

// from https://github.com/googleapis/google-cloud-go/blob/master/storage/go110.go
func isTransientGCSErr(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errutil.IsTransientErr(ctx, err) {
		return true
	}
	var googleErr *googleapi.Error
	if errors.As(err, &googleErr) {
		// Retry on 429 and 5xx, according to
		// https://cloud.google.com/storage/docs/exponential-backoff.
		return googleErr.Code == 429 || (googleErr.Code >= 500 && googleErr.Code < 600)
	}
	var tempErr interface{ Temporary() bool }
	if errors.As(err, &tempErr) {
		if tempErr.Temporary() {
			return true
		}
	}
	// Retry errors that might be an unexported type
	// Also picks up certain 500-level codes that are sent back from upstream gcp services
	// and not caught by the googleapi.Error case (Workload Identity in particular)
	retriable := []string{"connection refused", "connection reset", "Received 504",
		"Received 500", "TLS handshake timeout"}
	for _, s := range retriable {
		if strings.Contains(err.Error(), s) {
			return true
		}
	}
	if e, ok := err.(interface{ Unwrap() error }); ok {
		return isTransientGCSErr(ctx, e.Unwrap())
	}
	return false
}

func (h *ArtifactDriver) newGCSClient(ctx context.Context) (*storage.Client, error) {
	if h.ServiceAccountKey != "" {
		return newGCSClientWithCredential(ctx, h.ServiceAccountKey)
	}
	// Assume it uses Workload Identity
	return newGCSClientDefault(ctx)
}

func newGCSClientWithCredential(ctx context.Context, serviceAccountJSON string) (*storage.Client, error) {
	creds, err := google.CredentialsFromJSON(ctx, []byte(serviceAccountJSON), storage.ScopeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("GCS client CredentialsFromJSON: %w", err)
	}
	client, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("GCS storage.NewClient with credential: %w", err)
	}
	return client, nil
}

func newGCSClientDefault(ctx context.Context) (*storage.Client, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("GCS storage.NewClient: %w", err)
	}
	return client, nil
}

// Load function downloads objects from GCS
func (h *ArtifactDriver) Load(ctx context.Context, inputArtifact *wfv1.Artifact, path string) error {
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			key := filepath.Clean(inputArtifact.GCS.Key)
			logger := logging.RequireLoggerFromContext(ctx)
			logger.WithFields(logging.Fields{"path": path, "key": key}).Info(ctx, "GCS Load")
			gcsClient, err := h.newGCSClient(ctx)
			if err != nil {
				logger.WithError(err).Warn(ctx, "Failed to create new GCS client")
				return !isTransientGCSErr(ctx, err), err
			}
			defer gcsClient.Close()
			err = downloadObjects(ctx, gcsClient, inputArtifact.GCS.Bucket, key, path)
			if err != nil {
				logger.WithError(err).Warn(ctx, "Failed to download objects from GCS")
				return !isTransientGCSErr(ctx, err), err
			}
			return true, nil
		})
	return err
}

// download all the objects of a key from the bucket
func downloadObjects(ctx context.Context, client *storage.Client, bucket, key, path string) error {
	objNames, err := listByPrefix(ctx, client, bucket, key, "")
	if err != nil {
		return err
	}
	if len(objNames) < 1 {
		msg := fmt.Sprintf("no results for key: %s", key)
		return argoerrors.New(argoerrors.CodeNotFound, msg)
	}
	for _, objName := range objNames {
		err = downloadObject(ctx, client, bucket, key, objName, path)
		if err != nil {
			return err
		}
	}
	return nil
}

// download an object from the bucket
func downloadObject(ctx context.Context, client *storage.Client, bucket, key, objName, path string) error {
	objPrefix := filepath.Clean(key)
	if os.PathSeparator == '\\' {
		objPrefix = strings.ReplaceAll(objPrefix, "\\", "/")
	}

	relObjPath := strings.TrimPrefix(objName, objPrefix)
	localPath := filepath.Join(path, relObjPath)
	objectDir, _ := filepath.Split(localPath)
	if objectDir != "" {
		if err := os.MkdirAll(objectDir, 0o700); err != nil {
			return fmt.Errorf("mkdir %s: %w", objectDir, err)
		}
	}
	rc, err := client.Bucket(bucket).Object(objName).NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return argoerrors.New(argoerrors.CodeNotFound, err.Error())
		}
		return fmt.Errorf("new bucket reader: %w", err)
	}
	defer rc.Close()
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("os create %s: %w", localPath, err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			logger := logging.RequireLoggerFromContext(ctx)
			logger.WithField("path", localPath).WithError(err).Error(ctx, "Error closing file")
		}
	}()
	_, err = io.Copy(out, rc)
	if err != nil {
		return fmt.Errorf("io copy: %w", err)
	}
	return nil
}

// list all the object names of the prefix in the bucket
func listByPrefix(ctx context.Context, client *storage.Client, bucket, prefix, delim string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	it := client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: delim,
	})
	results := []string{}
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		// prefix is a file
		if attrs.Name == prefix {
			results = []string{attrs.Name}
			return results, nil
		}
		// skip "folder" path like objects
		// note that we still download content (including "subfolders")
		// this is just a consequence of how objects are stored in GCS (no real hierarchy)
		if strings.HasSuffix(attrs.Name, "/") {
			continue
		}
		results = append(results, attrs.Name)
	}
	return results, nil
}

func (h *ArtifactDriver) OpenStream(ctx context.Context, a *wfv1.Artifact) (io.ReadCloser, error) {
	// todo: this is a temporary implementation which loads file to disk first
	return common.LoadToStream(ctx, a, h)
}

// Save an artifact to GCS compliant storage, e.g., uploading a local file to GCS bucket
func (h *ArtifactDriver) Save(ctx context.Context, path string, outputArtifact *wfv1.Artifact) error {
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			key := filepath.Clean(outputArtifact.GCS.Key)
			logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"path": path, "key": key}).Info(ctx, "GCS Save")
			client, err := h.newGCSClient(ctx)
			if err != nil {
				return !isTransientGCSErr(ctx, err), err
			}
			defer client.Close()
			err = uploadObjects(ctx, client, outputArtifact.GCS.Bucket, key, path)
			if err != nil {
				return !isTransientGCSErr(ctx, err), err
			}
			return true, nil
		})
	return err
}

// list all the file relative paths under a dir
// path is suppoese to be a dir
// relPath is a given relative path to be inserted in front
func listFileRelPaths(path string, relPath string) ([]string, error) {
	results := []string{}
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			fs, err := listFileRelPaths(path+file.Name()+string(os.PathSeparator), relPath+file.Name()+string(os.PathSeparator))
			if err != nil {
				return nil, err
			}
			results = append(results, fs...)
		} else {
			results = append(results, relPath+file.Name())
		}
	}
	return results, nil
}

// upload a local file or dir to GCS
func uploadObjects(ctx context.Context, client *storage.Client, bucket, key, path string) error {
	isDir, err := file.IsDirectory(path)
	if err != nil {
		return fmt.Errorf("test if %s is a dir: %w", path, err)
	}
	if isDir {
		dirName := filepath.Clean(path) + string(os.PathSeparator)
		keyPrefix := filepath.Clean(key) + "/"
		fileRelPaths, err := listFileRelPaths(dirName, "")
		if err != nil {
			return err
		}
		for _, relPath := range fileRelPaths {
			fullKey := keyPrefix + relPath
			if os.PathSeparator == '\\' {
				fullKey = strings.ReplaceAll(fullKey, "\\", "/")
			}

			err = uploadObject(ctx, client, bucket, fullKey, dirName+relPath)
			if err != nil {
				return fmt.Errorf("upload %s: %w", dirName+relPath, err)
			}
		}
	} else {
		objectKey := filepath.Clean(key)
		if os.PathSeparator == '\\' {
			objectKey = strings.ReplaceAll(objectKey, "\\", "/")
		}
		err = uploadObject(ctx, client, bucket, objectKey, path)
		if err != nil {
			return fmt.Errorf("upload %s: %w", path, err)
		}
	}
	return nil
}

// upload an object to GCS
func uploadObject(ctx context.Context, client *storage.Client, bucket, key, localPath string) error {
	f, err := os.Open(filepath.Clean(localPath))
	if err != nil {
		return fmt.Errorf("os open: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logger := logging.RequireLoggerFromContext(ctx)
			logger.WithField("path", localPath).WithError(err).Error(ctx, "Error closing file")
		}
	}()
	wc := client.Bucket(bucket).Object(key).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return fmt.Errorf("io copy: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("writer close: %w", err)
	}
	return nil
}

// delete an object from GCS
func deleteObject(ctx context.Context, client *storage.Client, bucket, key string) error {
	err := client.Bucket(bucket).Object(key).Delete(ctx)
	if err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

// Delete deletes an artifact from GCS
func (h *ArtifactDriver) Delete(ctx context.Context, s *wfv1.Artifact) error {
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			client, err := h.newGCSClient(ctx)
			if err != nil {
				return !isTransientGCSErr(ctx, err), err
			}
			defer client.Close()
			err = deleteObject(ctx, client, s.GCS.Bucket, s.GCS.Key)
			if err != nil {
				return !isTransientGCSErr(ctx, err), err
			}
			return true, nil
		},
	)
	return err
}

func (h *ArtifactDriver) ListObjects(ctx context.Context, artifact *wfv1.Artifact) ([]string, error) {
	var files []string
	err := waitutil.Backoff(defaultRetry,
		func() (bool, error) {
			logger := logging.RequireLoggerFromContext(ctx)
			logger.WithFields(logging.Fields{"bucket": artifact.GCS.Bucket, "key": artifact.GCS.Key}).Info(ctx, "GCS List")
			client, err := h.newGCSClient(ctx)
			if err != nil {
				logger.WithError(err).Warn(ctx, "Failed to create new GCS client")
				return !isTransientGCSErr(ctx, err), err
			}
			defer client.Close()
			files, err = listByPrefix(ctx, client, artifact.GCS.Bucket, artifact.GCS.Key, "")
			if err != nil {
				return !isTransientGCSErr(ctx, err), err
			}
			return true, nil
		})
	return files, err
}

func (h *ArtifactDriver) IsDirectory(ctx context.Context, artifact *wfv1.Artifact) (bool, error) {
	return false, argoerrors.New(argoerrors.CodeNotImplemented, "IsDirectory currently unimplemented for GCS")
}
