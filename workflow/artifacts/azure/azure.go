package azure

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	log "github.com/sirupsen/logrus"

	argoerrors "github.com/argoproj/argo-workflows/v3/errors"

	"github.com/argoproj/pkg/file"
	"github.com/pkg/errors"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	artifactscommon "github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// ArtifactDriver is a driver for Azure Blob Storage
type ArtifactDriver struct {
	AccountKey  string
	Container   string
	Endpoint    string
	UseSDKCreds bool
}

var _ artifactscommon.ArtifactDriver = &ArtifactDriver{}

// newAzureContainerClient creates a new azblob.ContainerClient for interacting with the specified Azure Blob Storage container
// The container client is created with the default azblob.ClientOptions which does include retry behavior
// for failed requests.
func (azblobDriver *ArtifactDriver) newAzureContainerClient() (*azblob.ContainerClient, error) {

	containerUrl, err := url.Parse(azblobDriver.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Azure Blob Storage endpoint url %s: %s", azblobDriver.Endpoint, err)
	}
	// Append the container name to the URL path
	if len(containerUrl.Path) == 0 || containerUrl.Path[len(containerUrl.Path)-1] != '/' {
		containerUrl.Path += "/"
	}
	containerUrl.Path += azblobDriver.Container

	if azblobDriver.UseSDKCreds {
		credential, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("unable to create default Azure credential: %s", err)
		}
		containerClient, err := azblob.NewContainerClient(containerUrl.String(), credential, nil)
		return containerClient, err
	} else {
		if azblobDriver.AccountKey == "" {
			return nil, fmt.Errorf("accountKey secret is required for Azure Blob Storage if useSDKCreds is false")
		}
		accountName, err := determineAccountName(containerUrl)
		if err != nil {
			return nil, err
		}
		credential, err := azblob.NewSharedKeyCredential(accountName, azblobDriver.AccountKey)
		if err != nil {
			return nil, fmt.Errorf("unable to create Azure shared key credential: %s", err)
		}
		containerClient, err := azblob.NewContainerClientWithSharedKey(containerUrl.String(), credential, nil)
		return containerClient, err
	}
}

// determineAccountName determines the account name of the storage account based on the
// supplied container URL.
func determineAccountName(containerUrl *url.URL) (string, error) {
	hostname := containerUrl.Hostname()
	if strings.HasPrefix(hostname, "127.0.0.1") || strings.HasPrefix(hostname, "localhost") {
		parts := strings.Split(containerUrl.Path, "/")
		if len(parts) <= 2 {
			return "", errors.Errorf("unable to determine storage account name from %s", containerUrl)
		}
		return parts[1], nil
	} else {
		parts := strings.Split(hostname, ".")
		return parts[0], nil
	}
}

// Load downloads artifacts from Azure Blob Storage
func (azblobDriver *ArtifactDriver) Load(artifact *wfv1.Artifact, path string) error {
	log.WithFields(log.Fields{"endpoint": artifact.Azure.Endpoint, "container": artifact.Azure.Container,
		"blob": artifact.Azure.Blob}).Info("Downloading from Azure Blob Storage")
	containerClient, err := azblobDriver.newAzureContainerClient()
	if err != nil {
		return fmt.Errorf("unable to create Azure Blob Container client: %s", err)
	}

	// Assume we're not downloading a directory and try to download as a file, since this is
	// the most common case and we don't want the penalty of listing the blobs before we
	// download (to determine if it's a directory instead of a single file). If we get a
	// BlobNotFound error, then check if it's a directory and process accordingly. If the account
	// has HNS enabled (ADLS Gen 2), then there's an edge case with using the blob API to
	// access. The directory will be returned as an empty file, so check for that as well.
	var isEmptyFile bool
	origErr := DownloadFile(containerClient, artifact.Azure.Blob, path)
	if origErr == nil {
		fileInfo, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("unable to retrieve stats for downloaded file %s: %s", path, err)
		}

		// Empty file means it could be an ADLS Gen 2 account and we downloaded the
		// directory as an empty file -- we'll check below. If it's a non-empty file,
		// then we successfully downloaded a file blob.
		if fileInfo.Size() > 0 {
			return nil
		}
		isEmptyFile = true
	} else if !IsAzureError(origErr, azblob.StorageErrorCodeBlobNotFound) {
		return fmt.Errorf("unable to download blob %s: %s", artifact.Azure.Blob, origErr)
	}

	isDir, err := azblobDriver.IsDirectory(artifact)
	if err != nil {
		return fmt.Errorf("unable to determine if %s is a directory: %s", artifact.Azure.Blob, err)
	}

	// It's not a directory and the file doesn't exist, so return the original BlobNotFound error.
	if !isDir && !isEmptyFile {
		return fmt.Errorf("unable to download blob %s: %s", artifact.Azure.Blob, origErr)
	}

	// When we tried to download the blob as a file, we created an empty file for the
	// blob as a target. We need to delete that empty file so we can re-create as a directory.
	err = os.Remove(path)
	if err != nil {
		return fmt.Errorf("unable to remove attempted file download %s: %s", path, err)
	}

	// It's a directory, so download all of the files.
	err = azblobDriver.DownloadDirectory(containerClient, artifact, path)
	if err != nil {
		return fmt.Errorf("unable to download directory %s: %s", artifact.Azure.Blob, err)
	}

	return nil
}

// DownloadFile downloads a single file from Azure Blob Storage
func DownloadFile(containerClient *azblob.ContainerClient, blobName, path string) error {
	blobClient, err := containerClient.NewBlobClient(blobName)
	if err != nil {
		return fmt.Errorf("unable to create Azure Blob client for %s: %s", blobName, err)
	}

	outFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %s", path, err)
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Warnf("unable to close file: %s", err)
		}
	}()

	return blobClient.DownloadToFile(context.TODO(), 0, 0, outFile, azblob.DownloadOptions{})
}

// DownloadDirectory downloads all of the files starting with the named blob prefix into a local directory.
func (azblobDriver *ArtifactDriver) DownloadDirectory(containerClient *azblob.ContainerClient, artifact *wfv1.Artifact, path string) error {
	log.WithFields(log.Fields{"endpoint": artifact.Azure.Endpoint, "container": artifact.Azure.Container,
		"blob": artifact.Azure.Blob}).Info("Downloading directory from Azure Blob Storage")

	files, err := azblobDriver.ListObjects(artifact)
	if err != nil {
		return fmt.Errorf("unable to list blob %s in Azure Storage: %s", artifact.Azure.Blob, err)
	}

	err = os.Mkdir(path, 0755)
	if err != nil {
		return fmt.Errorf("unable to create local directory %s: %s", path, err)
	}

	for _, file := range files {
		// For ADLS Gen 2 accounts, we'll see a file whose name matches the directory. Skip it.
		if file == artifact.Azure.Blob {
			continue
		}

		relKeyPath := strings.TrimPrefix(file, artifact.Azure.Blob)
		localPath := filepath.Join(path, relKeyPath)

		err = DownloadFile(containerClient, file, localPath)
		if err != nil {
			return fmt.Errorf("unable to download file %s: %s", localPath, err)
		}
	}
	return nil
}

// OpenStream opens a stream reader for an artifact from Azure Blob Storage
func (azblobDriver *ArtifactDriver) OpenStream(artifact *wfv1.Artifact) (io.ReadCloser, error) {
	log.WithFields(log.Fields{"endpoint": artifact.Azure.Endpoint, "container": artifact.Azure.Container,
		"blob": artifact.Azure.Blob}).Info("Streaming from Azure Blob Storage")
	containerClient, err := azblobDriver.newAzureContainerClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create Azure Blob Container client: %s", err)
	}

	blobClient, err := containerClient.NewBlockBlobClient(artifact.Azure.Blob)
	if err != nil {
		return nil, fmt.Errorf("unable to create Azure Blob client for %s: %s", artifact.Azure.Blob, err)
	}

	// Attempt the download. If it fails with a BlobNotFound error, or succeeds but with
	// a content length of 0, then it could be that we're attempting to stream a directory.
	// Check if the blob represents a directory and return an error if so. If not, then
	// return either the original BlobNotFound error or the empty file stream.
	emptyFile := false
	response, origErr := blobClient.Download(context.TODO(), nil)
	if origErr == nil {
		emptyFile = *response.ContentLength == 0
		// We have a normal file blob, so just return the response body stream
		if !emptyFile {
			return response.Body(nil), nil
		}
	} else if !IsAzureError(origErr, azblob.StorageErrorCodeBlobNotFound) {
		return nil, fmt.Errorf("unable to open stream for blob %s: %s", artifact.Azure.Blob, origErr)
	}

	isDir, err := azblobDriver.IsDirectory(artifact)
	if err != nil {
		return nil, fmt.Errorf("unable to test if blob %s is a directory: %s", artifact.Azure.Blob, err)
	}
	if isDir {
		return nil, argoerrors.New(argoerrors.CodeNotImplemented, "Directory Stream capability currently unimplemented for Azure Blob")
	} else if !emptyFile {
		// Not a directory (and not successful retrieval of an empty file), so return
		// the original BlobNotFound error
		return nil, fmt.Errorf("unable to open blob stream for %s: %s", artifact.Azure.Blob, origErr)
	}

	return response.Body(nil), nil
}

// Save saves an artifact to Azure Blob Storage
func (azblobDriver *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	log.WithFields(log.Fields{"endpoint": outputArtifact.Azure.Endpoint, "container": outputArtifact.Azure.Container,
		"blob": outputArtifact.Azure.Blob}).Info("Saving to Azure Blob Storage")

	isDir, err := file.IsDirectory(path)
	if err != nil {
		return fmt.Errorf("failed to test if %s is a directory: %v", path, err)
	}

	containerClient, err := azblobDriver.newAzureContainerClient()
	if err != nil {
		return fmt.Errorf("unable to create Azure Blob Container client for %s: %s", outputArtifact.Azure.Blob, err)
	}

	if isDir {
		err := PutDirectory(containerClient, outputArtifact.Azure.Blob, path)
		if err != nil {
			return fmt.Errorf("unable to upload directory %s to Azure: %s", path, err)
		}
	} else {
		err := PutFile(containerClient, outputArtifact.Azure.Blob, path)
		if err != nil {
			return fmt.Errorf("unable to upload file %s to Azure: %s", path, err)
		}
	}

	return nil
}

// PutFile uploads a file to Azure Blob Storage
func PutFile(containerClient *azblob.ContainerClient, blobName, path string) error {
	blobClient, err := containerClient.NewBlockBlobClient(blobName)
	if err != nil {
		return fmt.Errorf("unable to create Azure Blob client: %s", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("unable to open file %s: %s", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Warnf("unable to close file %s: %s", path, err)
		}
	}()

	_, err = blobClient.UploadFile(context.TODO(), file, azblob.UploadOption{})
	return err
}

// PutDirectory uploads all files in a directory to Azure Blob Storage
func PutDirectory(containerClient *azblob.ContainerClient, blobName, path string) error {
	for putTask := range generatePutTasks(blobName, path) {
		err := PutFile(containerClient, putTask.blobName, putTask.path)
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes an artifact from a Azure Blob Storage
func (azblobDriver *ArtifactDriver) Delete(artifact *wfv1.Artifact) error {
	log.WithFields(log.Fields{"endpoint": artifact.Azure.Endpoint, "container": artifact.Azure.Container,
		"blob": artifact.Azure.Blob}).Info("Deleting object from Azure Blob Storage")
	containerClient, err := azblobDriver.newAzureContainerClient()
	if err != nil {
		return fmt.Errorf("unable to create Azure Blob Container client: %s", err)
	}

	isDir, err := azblobDriver.IsDirectory(artifact)
	if err != nil {
		return fmt.Errorf("unable to test if %s is a directory: %s", artifact.Azure.Blob, err)
	}

	if !isDir {
		return DeleteBlob(containerClient, artifact.Azure.Blob)
	} else {
		files, err := azblobDriver.ListObjects(artifact)
		if err != nil {
			return fmt.Errorf("unable to list files in %s: %s", artifact.Azure.Blob, err)
		}
		directoryFile := ""
		for _, file := range files {
			if file == artifact.Azure.Blob {
				directoryFile = file
				continue
			}

			if err := DeleteBlob(containerClient, file); err != nil {
				return err
			}
		}
		if directoryFile != "" {
			return DeleteBlob(containerClient, directoryFile)
		}
	}
	return nil
}

func DeleteBlob(containerClient *azblob.ContainerClient, blobName string) error {
	blobClient, err := containerClient.NewBlobClient(blobName)
	if err != nil {
		return fmt.Errorf("unable to create Azure Blob client for %s: %s", blobName, err)
	}

	_, err = blobClient.Delete(context.TODO(), nil)
	if err != nil {
		return fmt.Errorf("unable to delete Azure Blob %s: %s", blobName, err)
	}

	return err
}

// ListObjects lists the files in Azure Blob Storage
func (azblobDriver *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	var files []string
	log.WithFields(log.Fields{"endpoint": artifact.Azure.Endpoint, "container": artifact.Azure.Container,
		"blob": artifact.Azure.Blob}).Info("Listing blobs in Azure Blob Storage")

	containerClient, err := azblobDriver.newAzureContainerClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create Azure Blob Container client: %s", err)
	}

	listOpts := azblob.ContainerListBlobsFlatOptions{
		Prefix: &artifact.Azure.Blob,
		Marker: nil,
	}
	ctx := context.TODO()
	pager := containerClient.ListBlobsFlat(&listOpts)
	for pager.NextPage(ctx) {
		resp := pager.PageResponse()
		for _, v := range resp.Segment.BlobItems {
			files = append(files, *v.Name)
		}
	}
	if pager.Err() != nil {
		return nil, fmt.Errorf("error listing blobs %s in Azure Blob Storage container: %s", artifact.Azure.Blob, pager.Err())
	}
	return files, nil
}

// IsDirectory indicates whether or not the artifact represents a directory or a single file.
func (azblobDriver *ArtifactDriver) IsDirectory(artifact *wfv1.Artifact) (bool, error) {
	blobPrefix := artifact.Azure.Blob

	if blobPrefix == "" {
		return true, nil
	}
	if !strings.HasSuffix(blobPrefix, "/") {
		blobPrefix += "/"
	}

	containerClient, err := azblobDriver.newAzureContainerClient()
	if err != nil {
		return false, fmt.Errorf("unable to create Azure Blob Container client: %s", err)
	}

	listOpts := azblob.ContainerListBlobsFlatOptions{
		Prefix: &blobPrefix,
		Marker: nil,
	}
	pager := containerClient.ListBlobsFlat(&listOpts)
	if pager.NextPage(context.TODO()) {
		resp := pager.PageResponse()
		if len(resp.Segment.BlobItems) == 1 {
			return strings.HasPrefix(*resp.Segment.BlobItems[0].Name, blobPrefix), nil
		} else {
			return len(resp.Segment.BlobItems) > 0, nil
		}
	} else if pager.Err() != nil {
		return false, fmt.Errorf("error listing blobs %s in Azure Blob Storage container: %s", artifact.Azure.Blob, pager.Err())
	}

	return false, nil
}

func IsAzureError(err error, code azblob.StorageErrorCode) bool {
	var storageErr *azblob.StorageError
	if errors.As(err, &storageErr) {
		return storageErr.ErrorCode == code
	}
	return false
}

type uploadTask struct {
	blobName string
	path     string
}

func generatePutTasks(blobNamePrefix, rootPath string) chan uploadTask {
	rootPath = filepath.Clean(rootPath) + string(os.PathSeparator)
	uploadTasks := make(chan uploadTask)
	go func() {
		_ = filepath.Walk(rootPath, func(localPath string, fi os.FileInfo, _ error) error {
			relPath := strings.TrimPrefix(localPath, rootPath)
			if fi.IsDir() {
				return nil
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				return nil
			}
			t := uploadTask{
				blobName: path.Join(blobNamePrefix, relPath),
				path:     localPath,
			}
			uploadTasks <- t
			return nil
		})
		close(uploadTasks)
	}()
	return uploadTasks
}
