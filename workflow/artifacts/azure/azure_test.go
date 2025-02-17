package azure

import (
	"context"
	"errors"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/stretchr/testify/assert"
)

func TestDetermineAccountName(t *testing.T) {
	validUrls := []string{
		"https://accountname/",
		"https://accountname.blob.core.windows.net",
		"https://accountname.blob.core.windows.net/",
		"https://accountname.blob.core.windows.net:1234/",
		"https://localhost/accountname/foo",
		"https://127.0.0.1/accountname/foo",
		"https://localhost:1234/accountname/foo",
		"https://127.0.0.1:1234/accountname/foo",
	}
	for _, u := range validUrls {
		u, err := url.Parse(u)
		assert.NoError(t, err)
		accountName, err := determineAccountName(u)
		assert.NoError(t, err)
		assert.Equal(t, "accountname", accountName)
	}

	invalidUrls := []string{
		"https://127.0.0.1/foo",
	}
	for _, u := range invalidUrls {
		u, err := url.Parse(u)
		assert.NoError(t, err)
		accountName, err := determineAccountName(u)
		assert.Error(t, err)
		assert.Equal(t, "", accountName)
	}
}

func TestArtifactDriver_DownloadDirectory_Subdir(t *testing.T) {
	t.Skipf("This test needs azurite. docker run -p 10000:10000 mcr.microsoft.com/azure-storage/azurite:latest azurite-blob")

	driver := ArtifactDriver{
		AccountKey: "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==", // default azurite key
		Container:  "test",
		Endpoint:   "http://127.0.0.1:10000/devstoreaccount1",
	}

	// ensure container exists
	containerClient, err := driver.newAzureContainerClient()
	assert.NoError(t, err)
	_, err = containerClient.Create(context.Background(), nil)
	var responseError *azcore.ResponseError
	if err != nil && !(errors.As(err, &responseError) && responseError.ErrorCode == "ContainerAlreadyExists") {
		assert.NoError(t, err)
	}

	// put a file in a subdir on the azurite blob storage
	blobClient := containerClient.NewBlockBlobClient("dir/subdir/file-in-subdir.txt")
	_, err = blobClient.UploadBuffer(context.Background(), []byte("foo"), nil)
	assert.NoError(t, err)

	// download the dir, containing a subdir
	azureArtifact := wfv1.AzureArtifact{
		Blob: "dir",
	}
	argoArtifact := wfv1.Artifact{
		ArtifactLocation: wfv1.ArtifactLocation{
			Azure: &azureArtifact,
		},
	}
	dstDir := t.TempDir()
	err = driver.DownloadDirectory(containerClient, &argoArtifact, filepath.Join(dstDir, "dir"))
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(dstDir, "dir", "subdir", "file-in-subdir.txt"))
}
