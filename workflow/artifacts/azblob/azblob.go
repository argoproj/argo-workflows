package azblob

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	blob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/argoproj/pkg/file"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// ArtifactDriver is a driver for Azure Blob storage
type ArtifactDriver struct {
	AccountName string
	Container   string
	AccountKey  string
}

func (az *ArtifactDriver) newContainerURL() (*blob.ContainerURL, error) {
	if az.AccountKey == "" || az.AccountName == "" {
		return nil, fmt.Errorf("Either blog storage account name or account key is missing")
	}
	if az.Container == "" {
		return nil, fmt.Errorf("Container name is missing")
	}
	credential, err := blob.NewSharedKeyCredential(az.AccountName, az.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("new containerURL: %v", err)
	}
	pipeline := blob.NewPipeline(credential, blob.PipelineOptions{})
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", az.AccountName, az.Container))
	containerURL := blob.NewContainerURL(*URL, pipeline)
	return &containerURL, nil
}

// Load function downloads objects from Azure blob storage
func (az *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("Azure Blob Load path: %s, key: %s", path, inputArtifact.AzureBlob.Key)
			containerURL, err := az.newContainerURL()
			if err != nil {
				log.Warnf("Failed to create new Azure Blob containerURL: %v", err)
				return false, nil
			}
			err = downloadObjects(containerURL, inputArtifact.AzureBlob.Key, path)
			if err != nil {
				log.Warnf("Failed to download objects from Azure blog storage: %v", err)
				return false, nil
			}
			return true, nil
		})
	return err
}

// download all the objects of a key from the container
func downloadObjects(containerURL *blob.ContainerURL, key, localPath string) error {
	objNames, err := listByPrefix(containerURL, key)
	if err != nil {
		return fmt.Errorf("list oboject names: %v", err)
	}
	for _, objName := range objNames {
		err = downloadObject(containerURL, key, objName, localPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// list all the object names of the prefix in the container
func listByPrefix(containerURL *blob.ContainerURL, prefix string) ([]string, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	results := []string{}
	for marker := (blob.Marker{}); marker.NotDone(); {
		listBlob, err := containerURL.ListBlobsFlatSegment(ctx, marker, blob.ListBlobsSegmentOptions{Prefix: prefix})
		if err != nil {
			return nil, fmt.Errorf("ListBlobsFlatSegment error: %v", err)
		}
		marker = listBlob.NextMarker
		for _, blobInfo := range listBlob.Segment.BlobItems {
			results = append(results, blobInfo.Name)
		}
	}
	return results, nil
}

// download an object from the container
func downloadObject(containerURL *blob.ContainerURL, key, objName, path string) error {
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
	blobURL := containerURL.NewBlockBlobURL(objName)
	downloadResponse, err := blobURL.Download(ctx, 0, blob.CountToEnd, blob.BlobAccessConditions{}, false)
	if err != nil {
		return fmt.Errorf("download object: %v", err)
	}
	bodyStream := downloadResponse.Body(blob.RetryReaderOptions{MaxRetryRequests: 20})
	defer bodyStream.Close()
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %v", err)
	}
	defer out.Close()
	_, err = io.Copy(out, bodyStream)
	if err != nil {
		return fmt.Errorf("io copy: %v", err)
	}
	return nil
}

// Save an artifact to Azure Blob storage, e.g., uploading a local file to blob container
func (az *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			log.Infof("Azure Blob storage Save path: %s, key: %s", path, outputArtifact.AzureBlob.Key)
			containerURL, err := az.newContainerURL()
			if err != nil {
				log.Warnf("Failed to create new Azure Blob containerURL: %v", err)
				return false, nil
			}
			err = uploadObjects(containerURL, outputArtifact.AzureBlob.Key, path)
			if err != nil {
				log.Warnf("Failed to upload objects to Azure blob storage: %v", err)
			}
			return true, nil
		})
	return err
}

// upload a local file or dir to blob storage
func uploadObjects(containerURL *blob.ContainerURL, key, localPath string) error {
	isDir, err := file.IsDirectory(localPath)
	if err != nil {
		return fmt.Errorf("test if %s is a dir: %v", localPath, err)
	}
	if isDir {
		dirName := filepath.Clean(localPath) + "/"
		keyPrefix := filepath.Clean(key) + "/"
		fileRelPaths, err := listFileRelPaths(dirName, "")
		if err != nil {
			return err
		}
		for _, relPath := range fileRelPaths {
			err = uploadObject(containerURL, keyPrefix+relPath, dirName+relPath)
			if err != nil {
				return fmt.Errorf("upload %s: %v", dirName+relPath, err)
			}
		}
	} else {
		err = uploadObject(containerURL, filepath.Clean(key), localPath)
		if err != nil {
			return fmt.Errorf("upload %s: %v", localPath, err)
		}
	}
	return nil
}

// upload an object to blob storage
func uploadObject(containerURL *blob.ContainerURL, key, localFilePath string) error {
	blobURL := containerURL.NewBlockBlobURL(key)
	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("os.Open: %v", err)
	}
	defer file.Close()
	ctx := context.Background()
	_, err = blob.UploadFileToBlockBlob(ctx, file, blobURL, blob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16})
	if err != nil {
		return fmt.Errorf("file upload: %v", err)
	}
	return nil
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
