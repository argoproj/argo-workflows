package executor

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/mocks"
)

const (
	fakePodName       = "fake-test-pod-1234567890"
	fakeWorkflow      = "my-wf"
	fakeWorkflowUID   = "my-wf-uid"
	fakePodUID        = "my-pod-uid"
	fakeNodeID        = "my-node-id"
	fakeNamespace     = "default"
	fakeContainerName = "main"
)

func TestWorkflowExecutor_LoadArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		artifact wfv1.Artifact
		error    string
	}{
		{"ErrNotSupplied", wfv1.Artifact{Name: "foo"}, "required artifact 'foo' not supplied"},
		{"ErrFailedToLoad", wfv1.Artifact{
			Name: "foo",
			Path: "/tmp/foo.txt",
			ArtifactLocation: wfv1.ArtifactLocation{
				S3: &wfv1.S3Artifact{
					Key: "my-key",
				},
			},
		}, "failed to load artifact 'foo': template artifact location not set"},
		{"ErrNoPath", wfv1.Artifact{
			Name: "foo",
			ArtifactLocation: wfv1.ArtifactLocation{
				S3: &wfv1.S3Artifact{
					S3Bucket: wfv1.S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"},
					Key:      "my-key",
				},
			},
		}, "Artifact 'foo' did not specify a path"},
		{"ErrDirTraversal", wfv1.Artifact{
			Name: "foo",
			Path: "/tmp/../etc/passwd",
			ArtifactLocation: wfv1.ArtifactLocation{
				S3: &wfv1.S3Artifact{
					S3Bucket: wfv1.S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"},
					Key:      "my-key",
				},
			},
		}, "Artifact 'foo' attempted to use a path containing '..'. Directory traversal is not permitted"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			we := WorkflowExecutor{
				Template: wfv1.Template{
					Inputs: wfv1.Inputs{
						Artifacts: []wfv1.Artifact{test.artifact},
					},
				},
			}
			err := we.LoadArtifacts(ctx)
			require.EqualError(t, err, test.error)
		})
	}
}

func TestSaveParameters(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	templateWithOutParam := wfv1.Template{
		Outputs: wfv1.Outputs{
			Parameters: []wfv1.Parameter{
				{
					Name: "my-out",
					ValueFrom: &wfv1.ValueFrom{
						Path: "/path",
					},
				},
			},
		},
	}
	we := WorkflowExecutor{
		PodName:         fakePodName,
		Template:        templateWithOutParam,
		ClientSet:       fakeClientset,
		Namespace:       fakeNamespace,
		RuntimeExecutor: &mockRuntimeExecutor,
	}
	mockRuntimeExecutor.On("GetFileContents", fakeContainerName, "/path").Return("has a newline\n", nil)

	ctx := logging.TestContext(t.Context())
	err := we.SaveParameters(ctx)
	require.NoError(t, err)
	assert.Equal(t, "has a newline", we.Template.Outputs.Parameters[0].Value.String())
}

// TestIsBaseImagePath tests logic of isBaseImagePath which determines if a path is coming from a
// base image layer versus a shared volumeMount.
func TestIsBaseImagePath(t *testing.T) {
	templateWithSameDir := wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{
					Name: "samedir",
					Path: "/samedir",
				},
			},
		},
		Container: &corev1.Container{},
		Outputs: wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				{
					Name: "samedir",
					Path: "/samedir",
				},
			},
		},
	}

	we := WorkflowExecutor{
		Template: templateWithSameDir,
	}
	// 1. unrelated dir/file should be captured from base image layer
	assert.True(t, we.isBaseImagePath("/foo"))

	// 2. when input and output directory is same, it should be captured from shared emptyDir
	assert.False(t, we.isBaseImagePath("/samedir"))

	// 3. when output is a sub path of input dir, it should be captured from shared emptyDir
	we.Template.Outputs.Artifacts[0].Path = "/samedir/inner"
	assert.False(t, we.isBaseImagePath("/samedir/inner"))

	// 4. when output happens to overlap with input (in name only), it should be captured from base image layer
	we.Template.Inputs.Artifacts[0].Path = "/hello.txt"
	we.Template.Outputs.Artifacts[0].Path = "/hello.txt-COINCIDENCE"
	assert.True(t, we.isBaseImagePath("/hello.txt-COINCIDENCE"))

	// 5. when output is under a user specified volumeMount, it should be captured from shared mount
	we.Template.Inputs.Artifacts = nil
	we.Template.Container.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "workdir",
			MountPath: "/user-mount",
		},
	}
	we.Template.Outputs.Artifacts[0].Path = "/user-mount/some-path"
	assert.False(t, we.isBaseImagePath("/user-mount"))
	assert.False(t, we.isBaseImagePath("/user-mount/some-path"))
	assert.False(t, we.isBaseImagePath("/user-mount/some-path/foo"))
	assert.True(t, we.isBaseImagePath("/user-mount-coincidence"))
}

func TestDefaultParameters(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	templateWithOutParam := wfv1.Template{
		Outputs: wfv1.Outputs{
			Parameters: []wfv1.Parameter{
				{
					Name: "my-out",
					ValueFrom: &wfv1.ValueFrom{
						Default: wfv1.AnyStringPtr("Default Value"),
						Path:    "/path",
					},
				},
			},
		},
	}
	we := WorkflowExecutor{
		PodName:         fakePodName,
		Template:        templateWithOutParam,
		ClientSet:       fakeClientset,
		Namespace:       fakeNamespace,
		RuntimeExecutor: &mockRuntimeExecutor,
	}
	mockRuntimeExecutor.On("GetFileContents", fakeContainerName, "/path").Return("", fmt.Errorf("file not found"))

	ctx := logging.TestContext(t.Context())
	err := we.SaveParameters(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Default Value", we.Template.Outputs.Parameters[0].Value.String())
}

func TestDefaultParametersEmptyString(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	templateWithOutParam := wfv1.Template{
		Outputs: wfv1.Outputs{
			Parameters: []wfv1.Parameter{
				{
					Name: "my-out",
					ValueFrom: &wfv1.ValueFrom{
						Default: wfv1.AnyStringPtr(""),
						Path:    "/path",
					},
				},
			},
		},
	}
	we := WorkflowExecutor{
		PodName:         fakePodName,
		Template:        templateWithOutParam,
		ClientSet:       fakeClientset,
		Namespace:       fakeNamespace,
		RuntimeExecutor: &mockRuntimeExecutor,
	}
	mockRuntimeExecutor.On("GetFileContents", fakeContainerName, "/path").Return("", fmt.Errorf("file not found"))

	ctx := logging.TestContext(t.Context())
	err := we.SaveParameters(ctx)
	require.NoError(t, err)
	assert.Empty(t, we.Template.Outputs.Parameters[0].Value.String())
}

func TestIsTarball(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tests := []struct {
		path      string
		isTarball bool
		expectErr bool
	}{
		{"testdata/file", false, false},
		{"testdata/file.zip", false, false},
		{"testdata/file.tar", false, false},
		{"testdata/file.gz", false, false},
		{"testdata/file.tar.gz", true, false},
		{"testdata/file.tgz", true, false},
		{"testdata/not-found", false, true},
	}

	for _, test := range tests {
		ok, err := isTarball(ctx, test.path)
		if test.expectErr {
			require.Error(t, err, test.path)
		} else {
			require.NoError(t, err, test.path)
		}
		assert.Equal(t, test.isTarball, ok, test.path)
	}
}

func TestUnzip(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	zipPath := "testdata/file.zip"
	destPath := "testdata/unzippedFile"

	// test
	err := unzip(ctx, zipPath, destPath)
	require.NoError(t, err)

	// check unzipped file
	fileInfo, err := os.Stat(destPath)
	require.NoError(t, err)
	assert.True(t, fileInfo.Mode().IsRegular())

	// cleanup
	err = os.Remove(destPath)
	require.NoError(t, err)
}

func TestUntar(t *testing.T) {
	tarPath := "testdata/file.tar.gz"
	destPath := "testdata/untarredDir"
	filePath := "testdata/untarredDir/file"
	linkPath := "testdata/untarredDir/link"
	emptyDirPath := "testdata/untarredDir/empty-dir"

	// test
	err := untar(tarPath, destPath)
	require.NoError(t, err)

	// check untarred contents
	fileInfo, err := os.Stat(destPath)
	require.NoError(t, err)
	assert.True(t, fileInfo.Mode().IsDir())
	fileInfo, err = os.Stat(filePath)
	require.NoError(t, err)
	assert.True(t, fileInfo.Mode().IsRegular())
	dirInfo, err := os.Stat(destPath)
	require.NoError(t, err)
	// check that the modification time of the file is retained
	assert.True(t, fileInfo.ModTime().Before(dirInfo.ModTime()))
	fileInfo, err = os.Stat(linkPath)
	require.NoError(t, err)
	assert.True(t, fileInfo.Mode().IsRegular())
	fileInfo, err = os.Stat(emptyDirPath)
	require.NoError(t, err)
	assert.True(t, fileInfo.Mode().IsDir())

	// cleanup
	err = os.Remove(linkPath)
	require.NoError(t, err)
	err = os.Remove(filePath)
	require.NoError(t, err)
	err = os.Remove(emptyDirPath)
	require.NoError(t, err)
	err = os.Remove(destPath)
	require.NoError(t, err)
}

func TestChmod(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not work in windows")
	}

	type perm struct {
		dir  string
		file string
	}

	tests := []struct {
		mode        int32
		recurse     bool
		permissions perm
	}{
		{
			0o777,
			false,
			perm{
				"drwxrwxrwx",
				"-rw-------",
			},
		},
		{
			0o777,
			true,
			perm{
				"drwxrwxrwx",
				"-rwxrwxrwx",
			},
		},
	}

	for _, test := range tests {
		// Setup directory and file for testing
		tempDir := t.TempDir()

		tempFile, err := os.CreateTemp(tempDir, "chmod-file-test")
		require.NoError(t, err)

		// Run chmod function
		err = chmod(tempDir, test.mode, test.recurse)
		require.NoError(t, err)

		// Check directory mode if set
		dirPermission, err := os.Stat(tempDir)
		require.NoError(t, err)
		assert.Equal(t, dirPermission.Mode().String(), test.permissions.dir)

		// Check file mode mode if set
		filePermission, err := os.Stat(tempFile.Name())
		require.NoError(t, err)
		assert.Equal(t, filePermission.Mode().String(), test.permissions.file)
	}
}

func TestSaveArtifacts(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	mockTaskResultClient := argofake.NewSimpleClientset().ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace)
	templateWithOutParam := wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{
					Name: "samedir",
					Path: string(os.PathSeparator) + "samedir",
				},
			},
		},
		Outputs: wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				{
					Name:     "samedir",
					Path:     string(os.PathSeparator) + "samedir",
					Optional: true,
				},
			},
		},
	}
	templateOptionFalse := wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{
					Name: "samedir",
					Path: string(os.PathSeparator) + "samedir",
				},
			},
		},
		Outputs: wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				{
					Name:     "samedir",
					Path:     string(os.PathSeparator) + "samedir",
					Optional: false,
				},
			},
		},
	}
	templateZipArchive := wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{
					Name: "samedir",
					Path: string(os.PathSeparator) + "samedir",
				},
			},
		},
		Outputs: wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				{
					Name:     "samedir",
					Path:     string(os.PathSeparator) + "samedir",
					Optional: true,
					Archive: &wfv1.ArchiveStrategy{
						Zip: &wfv1.ZipStrategy{},
					},
				},
			},
		},
	}
	tests := []struct {
		workflowExecutor WorkflowExecutor
		expectError      bool
	}{
		{
			workflowExecutor: WorkflowExecutor{
				PodName:          fakePodName,
				Template:         templateWithOutParam,
				ClientSet:        fakeClientset,
				Namespace:        fakeNamespace,
				RuntimeExecutor:  &mockRuntimeExecutor,
				taskResultClient: mockTaskResultClient,
			},
			expectError: false,
		},
		{
			workflowExecutor: WorkflowExecutor{
				PodName:          fakePodName,
				Template:         templateOptionFalse,
				ClientSet:        fakeClientset,
				Namespace:        fakeNamespace,
				RuntimeExecutor:  &mockRuntimeExecutor,
				taskResultClient: mockTaskResultClient,
			},
			expectError: true,
		},
		{
			workflowExecutor: WorkflowExecutor{
				PodName:          fakePodName,
				Template:         templateZipArchive,
				ClientSet:        fakeClientset,
				Namespace:        fakeNamespace,
				RuntimeExecutor:  &mockRuntimeExecutor,
				taskResultClient: mockTaskResultClient,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		ctx := logging.TestContext(t.Context())
		_, err := tt.workflowExecutor.SaveArtifacts(ctx)
		if err != nil {
			assert.True(t, tt.expectError)
			continue
		}
		assert.False(t, tt.expectError)
	}
}

func TestMonitorProgress(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	annotationPackTickDuration := 5 * time.Millisecond
	readProgressFileTickDuration := time.Millisecond
	progressFile := "/tmp/progress"

	wfFake := argofake.NewSimpleClientset(&wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: fakeNamespace,
			Name:      fakeWorkflow,
		},
	})
	taskResults := wfFake.ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace)
	we := NewExecutor(
		ctx,
		nil,
		taskResults,
		nil,
		fakePodName,
		fakePodUID,
		fakeWorkflow,
		fakeWorkflowUID,
		fakeNodeID,
		fakeNamespace,
		&mocks.ContainerRuntimeExecutor{},
		wfv1.Template{},
		false,
		time.Now(),
		annotationPackTickDuration,
		readProgressFileTickDuration,
	)

	go we.monitorProgress(ctx, progressFile)

	err := os.WriteFile(progressFile, []byte("100/100\n"), os.ModePerm)
	require.NoError(t, err)

	time.Sleep(time.Second)

	result, err := taskResults.Get(ctx, fakeNodeID, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, fakeWorkflow, result.Labels[common.LabelKeyWorkflow])
	assert.Len(t, result.OwnerReferences, 1)
	assert.Equal(t, wfv1.Progress("100/100"), result.Progress)
}

func TestSaveLogs(t *testing.T) {
	const artStorageError = "artifact storage is not configured; see the docs for setup instructions: https://argo-workflows.readthedocs.io/en/latest/configure-artifact-repository/"
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	mockRuntimeExecutor.On("GetOutputStream", mock.Anything, mock.AnythingOfType("string"), true).Return(io.NopCloser(strings.NewReader("hello world")), nil)
	t.Run("Simple Pod node", func(t *testing.T) {
		templateWithArchiveLogs := wfv1.Template{
			ArchiveLocation: &wfv1.ArtifactLocation{
				ArchiveLogs: ptr.To(true),
			},
		}
		we := WorkflowExecutor{
			Template:        templateWithArchiveLogs,
			RuntimeExecutor: &mockRuntimeExecutor,
		}

		ctx := logging.TestContext(t.Context())
		logArtifacts := we.SaveLogs(ctx)

		require.EqualError(t, we.errors[0], artStorageError)
		assert.Empty(t, logArtifacts)
	})
}

func TestReportOutputs(t *testing.T) {
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	mockTaskResultClient := argofake.NewSimpleClientset().ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace)
	t.Run("Simple report output", func(t *testing.T) {
		artifacts := []wfv1.Artifact{
			{
				Name: "samedir",
				Path: "/samedir",
			},
		}
		templateWithArtifacts := wfv1.Template{
			Inputs: wfv1.Inputs{
				Artifacts: artifacts,
			},
		}
		we := WorkflowExecutor{
			Template:         templateWithArtifacts,
			RuntimeExecutor:  &mockRuntimeExecutor,
			taskResultClient: mockTaskResultClient,
		}

		ctx := logging.TestContext(t.Context())
		err := we.ReportOutputs(ctx, artifacts)

		require.NoError(t, err)
		assert.Empty(t, we.errors)
	})

}
