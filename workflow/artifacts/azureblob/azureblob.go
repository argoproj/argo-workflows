package azureblob

import (
	"github.com/Azure/azure-storage-blob-go/azblob"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type AzureBlobArtifactDriver struct {
	Endpoint    string
	Container   string
	AccountName string
	AccountKey  string
}

func (azblobDriver *AzureBlobArtifactDriver) newAzureBlobClient() (*azblob.ContainerURL, error) {
	_, _ = azblob.NewSharedKeyCredential(azblobDriver.AccountName, azblobDriver.AccountKey)
	panic("implement me")
}

func (azblobDriver *AzureBlobArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	panic("implement me")
}

func (azblobDriver *AzureBlobArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	panic("implement me")
}
