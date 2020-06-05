package azureblob

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type AzureBlobArtifactDriver struct {
	Endpoint    string
	Container   string
	AccountName string
	AccountKey  string
}

func (azblobDriver *AzureBlobArtifactDriver) newAzureBlobClient() (*azblob.ContainerURL, error) {
	credential, err := azblob.NewSharedKeyCredential(azblobDriver.AccountName, azblobDriver.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Azure credentials: %s", err)
	}

	pipeline := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	containerUrlString, err := url.Parse(fmt.Sprintf("https://%s/%s", azblobDriver.Endpoint, azblobDriver.Container))
	if err != nil {
		return nil, fmt.Errorf("unable to parse Azure URL: %s", err)
	}
	containerURL := azblob.NewContainerURL(*containerUrlString, pipeline)
	return &containerURL, nil
}

func (azblobDriver *AzureBlobArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			containerUrl, err := azblobDriver.newAzureBlobClient()
			if err != nil {
				return false, fmt.Errorf("unable to create Azure client: %s", err)
			}

			blobUrl := containerUrl.NewBlockBlobURL(inputArtifact.AzureBlob.Key)

			downloadResponse, err := blobUrl.Download(context.Background(), 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
			if err != nil {
				return false, fmt.Errorf("unable to download file from Azure: %s", err)
			}

			bodyStream := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})

			outFile, err := os.Create(path)
			if err != nil {
				return false, fmt.Errorf("unable to create file: %s", err)
			}
			defer func() {
				if err := outFile.Close(); err != nil {
					log.Warnf("unable to close file: %s", err)
				}
			}()

			if _, err = io.Copy(outFile, bodyStream); err != nil {
				return false, fmt.Errorf("unable to save file: %s", err)
			}

			return true, nil
		})
	return err
}

func (azblobDriver *AzureBlobArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	err := wait.ExponentialBackoff(wait.Backoff{Duration: time.Second * 2, Factor: 2.0, Steps: 5, Jitter: 0.1},
		func() (bool, error) {
			containerUrl, err := azblobDriver.newAzureBlobClient()
			if err != nil {
				return false, fmt.Errorf("unable to create Azure client: %s", err)
			}

			blobUrl := containerUrl.NewBlockBlobURL(outputArtifact.AzureBlob.Key)
			file, err := os.Open(path)

			_, err = azblob.UploadFileToBlockBlob(context.Background(), file, blobUrl, azblob.UploadToBlockBlobOptions{})
			if err != nil {
				return false, fmt.Errorf("unable to upload file to Azure: %s", err)
			}

			return true, nil
		})
	return err
}
