package azureblob

 import (
	"context"
	"fmt"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	log "github.com/sirupsen/logrus"
	"io"
	"net/url"
	"os"
)


 // AzureBlobArtifactDriver is a driver for Azure Blob Storage
type AzureBlobArtifactDriver struct {
	DefaultEndpointsProtocol string
	EndpointSuffix           string
	Container                string
	Key                      string
	AccountName              string
	AccountKey               string
}

 // newMinioClient instantiates a new minio client object.
func (azblobDriver *AzureBlobArtifactDriver) newAzureBlobClient() (*azblob.ContainerURL, error) {
   credentials,err := azblob.NewSharedKeyCredential(azblobDriver.AccountName, azblobDriver.AccountKey)
	if err != nil {
	log.Fatal("Invalid credentials with error: " + err.Error())
	return nil,err
    }

    p := azblob.NewPipeline(credentials, azblob.PipelineOptions{})

	u, err := url.Parse(fmt.Sprintf("https://%s.%s/%s", azblobDriver.AccountName,azblobDriver.Key,azblobDriver.Container))

	log.Info(azblobDriver.AccountName)
	log.Info(azblobDriver.Container)
	log.Info(azblobDriver.Key)

    if err != nil {
		return nil,err
	}

	containerUrl := azblob.NewContainerURL(*u, p)
	if err != nil {
		return nil,err
		}

 	return &containerUrl, nil
}

 // Load downloads artifacts from Azure Blob Storage
func (azblobDriver *AzureBlobArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {

    containerUrl,err := azblobDriver.newAzureBlobClient()
    if err != nil{
    return errors.InternalWrapError(err)
    }
	ctx := context.Background()
	log.Info(azblobDriver.EndpointSuffix)
	blobURL := containerUrl.NewBlockBlobURL(azblobDriver.EndpointSuffix)
    downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
    if err != nil {
		return errors.InternalWrapError(err)
	}
    bodyStream := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})
 	tmp, err := os.Create(path)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer tmp.Close()

 	if _, err = io.Copy(tmp, bodyStream); err != nil {
		return errors.InternalWrapError(err)
	}

 	return tmp.Sync()
}

 // Save  artifacts to Azure Blob Storage
func (azblobDriver *AzureBlobArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {

    containerUrl,err := azblobDriver.newAzureBlobClient()
    if err != nil{
    return errors.InternalWrapError(err)
    }
	ctx := context.Background()
	log.Info(azblobDriver.EndpointSuffix)
	blobURL := containerUrl.NewBlockBlobURL(azblobDriver.EndpointSuffix)

 	file, err := os.Open(path)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer file.Close()

 	r, err := azblob.UploadFileToBlockBlob(ctx, file, blobURL,azblob.UploadToBlockBlobOptions{
	BlockSize:   4 * 1024 * 1024,
	Parallelism: 16})
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Info(r)

 	return nil
}