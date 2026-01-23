package azure

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.NoError(t, err)
		accountName, err := determineAccountName(u)
		require.NoError(t, err)
		assert.Equal(t, "accountname", accountName)
	}

	invalidUrls := []string{
		"https://127.0.0.1/foo",
	}
	for _, u := range invalidUrls {
		u, err := url.Parse(u)
		require.NoError(t, err)
		accountName, err := determineAccountName(u)
		require.Error(t, err)
		assert.Empty(t, accountName)
	}
}

func TestArtifactDriver_WithServiceKey_DownloadDirectory_Subdir(t *testing.T) {
	t.Skipf("This test needs azurite. docker run -p 10000:10000 mcr.microsoft.com/azure-storage/azurite:latest azurite-blob")

	driver := ArtifactDriver{
		AccountKey: "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==", // default azurite key
		Container:  "test",
		Endpoint:   "http://127.0.0.1:10000/devstoreaccount1",
	}

	// ensure container exists
	ctx := logging.TestContext(t.Context())
	containerClient, err := driver.newAzureContainerClient(ctx)
	require.NoError(t, err)
	_, err = containerClient.Create(ctx, nil)
	var responseError *azcore.ResponseError
	if err != nil && (!errors.As(err, &responseError) || responseError.ErrorCode != "ContainerAlreadyExists") {
		require.NoError(t, err)
	}

	// test read/write operations to the azurite container  using the container client
	testContainerClientReadWriteOperations(t, containerClient, driver)
}

func TestArtifactDriver_WithSASToken_DownloadDirectory_Subdir(t *testing.T) {
	t.Skipf("This test needs azurite. docker run -p 10000:10000 mcr.microsoft.com/azure-storage/azurite:latest azurite-blob")

	driver := ArtifactDriver{
		AccountKey: "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==", // default azurite key
		Container:  "test",
		Endpoint:   "http://127.0.0.1:10000/devstoreaccount1",
	}

	containerURL, _ := url.Parse(driver.Endpoint)
	if len(containerURL.Path) == 0 || containerURL.Path[len(containerURL.Path)-1] != '/' {
		containerURL.Path += "/"
	}
	containerURL.Path += driver.Container

	accountName, _ := determineAccountName(containerURL)
	credential, _ := azblob.NewSharedKeyCredential(accountName, driver.AccountKey)

	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPSandHTTP,
		StartTime:     time.Now().UTC().Add(time.Second * -10),
		ExpiryTime:    time.Now().UTC().Add(15 * time.Minute),
		Permissions:   to.Ptr(sas.ContainerPermissions{Read: true, Write: true, List: true}).String(),
		ContainerName: driver.Container,
	}.SignWithSharedKey(credential)
	if err != nil {
		t.Fatalf("Failed to create SAS query params: %s", err.Error())
	}

	driver.AccountKey = sasQueryParams.Encode()

	// ensure container exists
	ctx := logging.TestContext(t.Context())
	containerClient, err := driver.newAzureContainerClient(ctx)
	require.NoError(t, err)

	// test read/write operations to the azurite container  using the container client
	testContainerClientReadWriteOperations(t, containerClient, driver)

}

func testContainerClientReadWriteOperations(t *testing.T, containerClient *container.Client, driver ArtifactDriver) {
	ctx := logging.TestContext(t.Context())
	// put a file in a subdir on the azurite blob storage
	// download the dir, containing a subdir
	blobClient := containerClient.NewBlockBlobClient("dir/subdir/file-in-subdir.txt")
	_, err := blobClient.UploadBuffer(ctx, []byte("foo"), nil)
	require.NoError(t, err)

	azureArtifact := wfv1.AzureArtifact{
		Blob: "dir",
	}
	argoArtifact := wfv1.Artifact{
		ArtifactLocation: wfv1.ArtifactLocation{
			Azure: &azureArtifact,
		},
	}
	dstDir := t.TempDir()
	err = driver.DownloadDirectory(ctx, containerClient, &argoArtifact, filepath.Join(dstDir, "dir"))
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dstDir, "dir", "subdir", "file-in-subdir.txt"))
}

func TestIsSASAccountKey(t *testing.T) {
	// Define test cases
	testCases := []struct {
		accountKey string
		expected   bool
	}{
		// Valid SAS tokens
		{"?sv=2019-12-12&ss=b&srt=sco&sp=rwdlacupx&se=2021-12-12T00:00:00Z&st=2021-01-01T00:00:00Z&spr=https&sig=signature", true},
		{"?sv=2020-08-04&ss=b&srt=sco&sp=rwdlacupx&se=2022-12-12T00:00:00Z&st=2021-01-01T00:00:00Z&spr=https&sig=signature", true},
		// Invalid SAS tokens
		{"Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==", false},
		{"invalid-sas-token", false},
	}

	for _, tc := range testCases {
		t.Run(tc.accountKey, func(t *testing.T) {
			result := isSASAccountKey(tc.accountKey)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGeneratePutTasks tests the generatePutTasks function
func TestGeneratePutTasks(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0600)
	require.NoError(t, err)

	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0600)
	require.NoError(t, err)

	// Test generatePutTasks
	tasks := generatePutTasks("prefix", tempDir)

	var taskList []uploadTask
	for task := range tasks {
		taskList = append(taskList, task)
	}

	assert.Len(t, taskList, 2)

	// Check that both files are in the task list
	hasFile1 := false
	hasFile2 := false
	for _, task := range taskList {
		if strings.HasSuffix(task.path, "file1.txt") {
			hasFile1 = true
			assert.Equal(t, "prefix/file1.txt", task.blobName)
		}
		if strings.HasSuffix(task.path, "file2.txt") {
			hasFile2 = true
			assert.Equal(t, "prefix/subdir/file2.txt", task.blobName)
		}
	}
	assert.True(t, hasFile1, "file1.txt should be in tasks")
	assert.True(t, hasFile2, "subdir/file2.txt should be in tasks")
}

// TestGeneratePutTasksEmptyDir tests generatePutTasks with an empty directory
func TestGeneratePutTasksEmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	tasks := generatePutTasks("prefix", tempDir)

	var taskList []uploadTask
	for task := range tasks {
		taskList = append(taskList, task)
	}

	assert.Len(t, taskList, 0)
}

// TestArtifactDriverMissingAccountKey tests that SaveStream fails without credentials
func TestArtifactDriverMissingAccountKey(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	driver := ArtifactDriver{
		AccountKey:  "", // No account key
		Container:   "test",
		Endpoint:    "http://127.0.0.1:10000/devstoreaccount1",
		UseSDKCreds: false,
	}

	_, err := driver.newAzureContainerClient(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accountKey secret is required")
}
