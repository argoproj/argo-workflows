package gcs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/pkg/file"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// ArtifactDriver is a driver for GCS
type ArtifactDriver struct {
	ServiceAccountKey string
}

func (g *ArtifactDriver) newGCSClient() (*storage.Client, error) {
	if g.ServiceAccountKey != "" {
		return newGCSClientWithCredential(g.ServiceAccountKey)
	}
	// Assume it uses Workload Identity
	return newGCSClientDefault()
}

func newGCSClientWithCredential(serviceAccountJSON string) (*storage.Client, error) {
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(serviceAccountJSON), storage.ScopeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("GCS client CredentialsFromJSON: %v", err)
	}
	client, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("GCS storage.NewClient with credential: %v", err)
	}
	return client, nil
}

func newGCSClientDefault() (*storage.Client, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("GCS storage.NewClient: %v", err)
	}
	return client, nil
}

// Load function downloads objects from GCS
func (g *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("GCS Load path: %s, key: %s", path, inputArtifact.GCS.Key)
			gcsClient, err := g.newGCSClient()
			if err != nil {
				log.Warnf("Failed to create new GCS client: %v", err)
				return false, err
			}
			defer gcsClient.Close()
			err = downloadObjects(gcsClient, inputArtifact.GCS.Bucket, inputArtifact.GCS.Key, path)
			if err != nil {
				log.Warnf("Failed to download objects from GCS: %v", err)
				return false, err
			}
			return true, nil
		})
	return err
}

// download all the objects of a key from the bucket
func downloadObjects(client *storage.Client, bucket, key, path string) error {
	objNames, err := listByPrefix(client, bucket, key, "")
	if err != nil {
		return err
	}
	for _, objName := range objNames {
		err = downloadObject(client, bucket, key, objName, path)
		if err != nil {
			return err
		}
	}
	return nil
}

// download an object from the bucket
func downloadObject(client *storage.Client, bucket, key, objName, path string) error {
	objPrefix := filepath.Clean(key)
	relObjPath := strings.TrimPrefix(objName, objPrefix)
	localPath := filepath.Join(path, relObjPath)
	objectDir, _ := filepath.Split(localPath)
	if objectDir != "" {
		if err := os.MkdirAll(objectDir, 0700); err != nil {
			return fmt.Errorf("mkdir %s: %v", objectDir, err)
		}
	}
	ctx := context.Background()
	rc, err := client.Bucket(bucket).Object(objName).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return errors.New(errors.CodeNotFound, err.Error())
		}
		return fmt.Errorf("new bucket reader: %v", err)
	}
	defer rc.Close()
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("os create %s: %v", localPath, err)
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	if err != nil {
		return fmt.Errorf("io copy: %v", err)
	}
	return nil
}

// list all the object names of the prefix in the bucket
func listByPrefix(client *storage.Client, bucket, prefix, delim string) ([]string, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	it := client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: delim,
	})
	results := []string{}
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		results = append(results, attrs.Name)
	}
	return results, nil
}

// Save an artifact to GCS compliant storage, e.g., uploading a local file to GCS bucket
func (g *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("GCS Save path: %s, key: %s", path, outputArtifact.GCS.Key)
			client, err := g.newGCSClient()
			if err != nil {
				return false, err
			}
			defer client.Close()
			err = uploadObjects(client, outputArtifact.GCS.Bucket, outputArtifact.GCS.Key, path)
			if err != nil {
				return false, err
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
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			fs, err := listFileRelPaths(path+file.Name()+"/", relPath+file.Name()+"/")
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
func uploadObjects(client *storage.Client, bucket, key, path string) error {
	isDir, err := file.IsDirectory(path)
	if err != nil {
		return fmt.Errorf("test if %s is a dir: %v", path, err)
	}
	if isDir {
		dirName := filepath.Clean(path) + "/"
		keyPrefix := filepath.Clean(key) + "/"
		fileRelPaths, err := listFileRelPaths(dirName, "")
		if err != nil {
			return err
		}
		for _, relPath := range fileRelPaths {
			err = uploadObject(client, bucket, keyPrefix+relPath, dirName+relPath)
			if err != nil {
				return fmt.Errorf("upload %s: %v", dirName+relPath, err)
			}
		}
	} else {
		err = uploadObject(client, bucket, filepath.Clean(key), path)
		if err != nil {
			return fmt.Errorf("upload %s: %v", path, err)
		}
	}
	return nil
}

// upload an object to GCS
func uploadObject(client *storage.Client, bucket, key, localPath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("os open: %v", err)
	}
	defer f.Close()
	ctx := context.Background()
	wc := client.Bucket(bucket).Object(key).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return fmt.Errorf("io copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("writer close: %v", err)
	}
	return nil
}
