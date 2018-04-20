package azureblob

import (
	"context"
	"fmt"
	"github.com/Azure/azure-storage-blob-go/2017-07-29/azblob"
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
func (azblobDriver *AzureBlobArtifactDriver) newAzureBlobClient() (*azblob.ServiceURL, error) {
	credentials := azblob.NewSharedKeyCredential(azblobDriver.AccountName, azblobDriver.AccountKey)
	p := azblob.NewPipeline(credentials, azblob.PipelineOptions{})
	u, err := url.Parse(fmt.Sprintf("https://%s.%s", azblobDriver.AccountName, azblobDriver.EndpointSuffix))
	if err != nil {
		return nil, err
	}
	serviceURL := azblob.NewServiceURL(*u, p)

	return &serviceURL, nil
}

// Load downloads artifacts from Azure Blob Storage
func (azblobDriver *AzureBlobArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	svcURL, err := azblobDriver.newAzureBlobClient()
	if err != nil {
		return errors.InternalWrapError(err)
	}
	ctx := context.Background()
	cntURL := svcURL.NewContainerURL(azblobDriver.Container)
	blockBlobURL := cntURL.NewBlockBlobURL(azblobDriver.Key)

	r, err := blockBlobURL.GetBlob(ctx, azblob.BlobRange{}, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer r.Body().Close()

	tmp, err := os.Create(path)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer tmp.Close()

	if _, err = io.Copy(tmp, r.Body()); err != nil {
		return errors.InternalWrapError(err)
	}

	return tmp.Sync()
}

// Save  artifacts to Azure Blob Storage
func (azblobDriver *AzureBlobArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	svcURL, err := azblobDriver.newAzureBlobClient()
	if err != nil {
		return errors.InternalWrapError(err)
	}

	ctx := context.Background()
	cntURL := svcURL.NewContainerURL(azblobDriver.Container)
	blockBlobURL := cntURL.NewBlockBlobURL(path)

	file, err := os.Open(path)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer file.Close()

	r, err := blockBlobURL.PutBlob(ctx, file, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Infof(r.Status())

	return nil
}
