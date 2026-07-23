package executor

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
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

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/mocks"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
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
			tracing, err := tracing.New(ctx, `argoexec`) // TODO arguments here
			require.NoError(t, err)
			we := WorkflowExecutor{
				Template: wfv1.Template{
					Inputs: wfv1.Inputs{
						Artifacts: []wfv1.Artifact{test.artifact},
					},
				},
				Tracing: tracing,
			}
			err = we.loadArtifacts(ctx, "")
			require.EqualError(t, err, test.error)
		})
	}
}

func TestSaveParameters(t *testing.T) {
	fakeClientset := fake.NewClientset()
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

// TestIsBaseImagePathInitless covers the init-less divergence in isBaseImagePath:
// when an output path overlaps an input artifact path, init-less mode must treat
// it as a base image path so the output is read from the emissary-staged live
// file rather than the shared input emptyDir (which would silently upload the
// stale input when the user replaces the file via rm+recreate or rename). Legacy
// mode keeps reading the shared mount, where the SubPath bind mount stays current.
func TestIsBaseImagePathInitless(t *testing.T) {
	newWe := func() *WorkflowExecutor {
		return &WorkflowExecutor{
			Template: wfv1.Template{
				Container: &corev1.Container{},
				Inputs: wfv1.Inputs{
					Artifacts: []wfv1.Artifact{{Name: "samedir", Path: "/samedir"}},
				},
				Outputs: wfv1.Outputs{
					Artifacts: []wfv1.Artifact{{Name: "samedir", Path: "/samedir"}},
				},
			},
		}
	}

	t.Run("legacy mode reads the shared input mount", func(t *testing.T) {
		we := newWe()
		// Exact overlap and sub-path overlap both come from the shared emptyDir.
		assert.False(t, we.isBaseImagePath("/samedir"))
		we.Template.Outputs.Artifacts[0].Path = "/samedir/inner"
		assert.False(t, we.isBaseImagePath("/samedir/inner"))
	})

	t.Run("init-less mode reads the live emissary-staged output", func(t *testing.T) {
		t.Setenv(common.EnvVarInitlessPod, "true")
		we := newWe()
		assert.True(t, we.isBaseImagePath("/samedir"))
		we.Template.Outputs.Artifacts[0].Path = "/samedir/inner"
		assert.True(t, we.isBaseImagePath("/samedir/inner"))
	})

	t.Run("init-less mode still reads user volumes from the mirror", func(t *testing.T) {
		t.Setenv(common.EnvVarInitlessPod, "true")
		we := newWe()
		// A user-declared volume overlap is delivered into the real (shared) volume
		// and the emissary intentionally skips staging it, so it must keep reading
		// the mirror even in init-less mode.
		we.Template.Inputs.Artifacts = nil
		we.Template.Container.VolumeMounts = []corev1.VolumeMount{{Name: "workdir", MountPath: "/user-mount"}}
		we.Template.Outputs.Artifacts[0].Path = "/user-mount/some-path"
		assert.False(t, we.isBaseImagePath("/user-mount/some-path"))
	})
}

// TestStageArchiveFileInitlessOverlap proves the read-side fix end to end: with
// an input artifact and an output artifact sharing a path, init-less staging
// fetches the live output through the runtime executor (the emissary already
// tarred main's current file), while legacy staging reads the mirrored mount and
// never touches the live-output path.
func TestStageArchiveFileInitlessOverlap(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tr, err := tracing.New(ctx, "argoexec")
	require.NoError(t, err)

	newWe := func(rt *mocks.ContainerRuntimeExecutor) *WorkflowExecutor {
		return &WorkflowExecutor{
			Template: wfv1.Template{
				Inputs:  wfv1.Inputs{Artifacts: []wfv1.Artifact{{Name: "samedir", Path: "/samedir"}}},
				Outputs: wfv1.Outputs{Artifacts: []wfv1.Artifact{{Name: "samedir", Path: "/samedir"}}},
			},
			RuntimeExecutor: rt,
			Tracing:         tr,
		}
	}

	t.Run("init-less fetches the live output via the runtime executor", func(t *testing.T) {
		t.Setenv(common.EnvVarInitlessPod, "true")
		rt := &mocks.ContainerRuntimeExecutor{}
		// CopyFile is the live-output path: the emissary inside main has already
		// staged the current (possibly rm+recreated) file. Reading the input
		// emptyDir instead would skip CopyFile and upload the stale input.
		rt.On("CopyFile", mock.Anything, common.MainContainerName, "/samedir", mock.Anything, mock.Anything).Return(nil)
		we := newWe(rt)
		art := we.Template.Outputs.Artifacts[0]
		fileName, localArtPath, err := we.stageArchiveFile(ctx, common.MainContainerName, &art)
		require.NoError(t, err)
		assert.Equal(t, "samedir.tgz", fileName)
		assert.NotEmpty(t, localArtPath)
		rt.AssertCalled(t, "CopyFile", mock.Anything, common.MainContainerName, "/samedir", mock.Anything, mock.Anything)
	})

	t.Run("legacy reads the mirrored mount, never the live-output path", func(t *testing.T) {
		rt := &mocks.ContainerRuntimeExecutor{}
		we := newWe(rt)
		art := we.Template.Outputs.Artifacts[0]
		// Legacy staging reads /mainctrfs/samedir directly. There is no such file in
		// this unit test so staging fails, but the point is that the live-output
		// path (CopyFile) is never used in legacy mode.
		_, _, _ = we.stageArchiveFile(ctx, common.MainContainerName, &art)
		rt.AssertNotCalled(t, "CopyFile")
	})
}

func TestDefaultParameters(t *testing.T) {
	fakeClientset := fake.NewClientset()
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
	fakeClientset := fake.NewClientset()
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
	fakeClientset := fake.NewClientset()
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	mockTaskResultClient := argofake.NewClientset().ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace)
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
	ctx := logging.TestContext(t.Context())
	tracing, err := tracing.New(ctx, `argoexec`) // TODO arguments here
	require.NoError(t, err)
	tests := []struct {
		workflowExecutor *WorkflowExecutor
		expectError      bool
	}{
		{
			workflowExecutor: &WorkflowExecutor{
				PodName:          fakePodName,
				Template:         templateWithOutParam,
				ClientSet:        fakeClientset,
				Namespace:        fakeNamespace,
				RuntimeExecutor:  &mockRuntimeExecutor,
				taskResultClient: mockTaskResultClient,
				Tracing:          tracing,
			},
			expectError: false,
		},
		{
			workflowExecutor: &WorkflowExecutor{
				PodName:          fakePodName,
				Template:         templateOptionFalse,
				ClientSet:        fakeClientset,
				Namespace:        fakeNamespace,
				RuntimeExecutor:  &mockRuntimeExecutor,
				taskResultClient: mockTaskResultClient,
				Tracing:          tracing,
			},
			expectError: true,
		},
		{
			workflowExecutor: &WorkflowExecutor{
				PodName:          fakePodName,
				Template:         templateZipArchive,
				ClientSet:        fakeClientset,
				Namespace:        fakeNamespace,
				RuntimeExecutor:  &mockRuntimeExecutor,
				taskResultClient: mockTaskResultClient,
				Tracing:          tracing,
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

	wfFake := argofake.NewClientset(&wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: fakeNamespace,
			Name:      fakeWorkflow,
		},
	})
	taskResults := wfFake.ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace)
	we, err := NewExecutor(
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
	require.NoError(t, err)

	go we.monitorProgress(ctx, progressFile)

	err = os.WriteFile(progressFile, []byte("100/100\n"), os.ModePerm)
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
	templateWithMainAndSystemLogs := wfv1.Template{
		ArchiveLocation: &wfv1.ArtifactLocation{
			ArchiveLogs:                new(true),
			ArchiveSystemContainerLogs: new(true),
		},
	}
	t.Run("Simple Pod node", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		tracing, err := tracing.New(ctx, `argoexec`)
		require.NoError(t, err)

		mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, mock.AnythingOfType("string"), true).Return(io.NopCloser(strings.NewReader("hello world")), nil)

		templateWithArchiveLogs := wfv1.Template{
			ArchiveLocation: &wfv1.ArtifactLocation{
				ArchiveLogs: new(true),
			},
		}
		we := WorkflowExecutor{
			Template:        templateWithArchiveLogs,
			RuntimeExecutor: &mockRuntimeExecutor,
			Tracing:         tracing,
		}

		logArtifacts := we.SaveLogs(ctx)

		require.EqualError(t, we.errors[0], artStorageError)
		assert.Empty(t, logArtifacts)
	})

	t.Run("init and wait not archived when flag is false", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		tr, err := tracing.New(ctx, `argoexec`)
		require.NoError(t, err)

		mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "main", true).Return(io.NopCloser(strings.NewReader("main log")), nil)

		templateWithArchiveLogsOnly := wfv1.Template{
			ArchiveLocation: &wfv1.ArtifactLocation{
				ArchiveLogs: new(true),
				// ArchiveSystemContainerLogs not set (nil)
			},
		}
		we := WorkflowExecutor{
			Template:        templateWithArchiveLogsOnly,
			RuntimeExecutor: &mockRuntimeExecutor,
			Tracing:         tr,
		}

		we.SaveLogs(ctx)

		mockRuntimeExecutor.AssertCalled(t, "GetOutputStream", mock.Anything, "main", true)
		mockRuntimeExecutor.AssertNotCalled(t, "GetOutputStream", mock.Anything, "init", true)
		mockRuntimeExecutor.AssertNotCalled(t, "GetOutputStream", mock.Anything, "wait", true)
	})

	t.Run("init and wait archived even when archiveLogs is false", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		tr, err := tracing.New(ctx, `argoexec`)
		require.NoError(t, err)

		mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
		// main must not be called (archiveLogs is false)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "init", true).Return(io.NopCloser(strings.NewReader("init log")), nil)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "wait", true).Return(io.NopCloser(strings.NewReader("wait log")), nil)

		templateSystemOnly := wfv1.Template{
			ArchiveLocation: &wfv1.ArtifactLocation{
				// ArchiveLogs not set (nil)
				ArchiveSystemContainerLogs: new(true),
			},
		}

		we := WorkflowExecutor{
			Template:        templateSystemOnly,
			RuntimeExecutor: &mockRuntimeExecutor,
			Tracing:         tr,
		}

		we.SaveLogs(ctx)

		mockRuntimeExecutor.AssertNotCalled(t, "GetOutputStream", mock.Anything, "main", true)
		mockRuntimeExecutor.AssertCalled(t, "GetOutputStream", mock.Anything, "init", true)
		mockRuntimeExecutor.AssertCalled(t, "GetOutputStream", mock.Anything, "wait", true)
	})

	t.Run("init is processed and wait is skipped when its combined file is missing", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		tr, err := tracing.New(ctx, `argoexec`)
		require.NoError(t, err)

		mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "main", true).Return(io.NopCloser(strings.NewReader("main log")), nil)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "init", true).Return(io.NopCloser(strings.NewReader("init log")), nil)
		// wait log file does not exist (simulating tee setup failure)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "wait", true).Return(nil, fs.ErrNotExist)

		we := WorkflowExecutor{
			Template:        templateWithMainAndSystemLogs,
			RuntimeExecutor: &mockRuntimeExecutor,
			Tracing:         tr,
		}

		we.SaveLogs(ctx)

		mockRuntimeExecutor.AssertCalled(t, "GetOutputStream", mock.Anything, "main", true)
		mockRuntimeExecutor.AssertCalled(t, "GetOutputStream", mock.Anything, "init", true)
		mockRuntimeExecutor.AssertCalled(t, "GetOutputStream", mock.Anything, "wait", true)

		require.Len(t, we.errors, 2, "main and init recorded as storage errors, wait skipped")

		for _, err := range we.errors {
			assert.EqualError(t, err, artStorageError)
		}
	})

	t.Run("skip init and wait when combined file does not exist", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		tr, err := tracing.New(ctx, `argoexec`)
		require.NoError(t, err)

		mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "main", true).Return(io.NopCloser(strings.NewReader("main log")), nil)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "init", true).Return(nil, fs.ErrNotExist)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "wait", true).Return(nil, fs.ErrNotExist)

		we := WorkflowExecutor{
			Template:        templateWithMainAndSystemLogs,
			RuntimeExecutor: &mockRuntimeExecutor,
			Tracing:         tr,
		}

		we.SaveLogs(ctx)

		// exactly one error (main's storage error); init / wait are skipped on
		// ErrNotExist and must not record any error.
		require.Len(t, we.errors, 1, "only the main container's storage error is expected")
		assert.EqualError(t, we.errors[0], artStorageError)
	})

	t.Run("wait logs are saved last", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		tr, err := tracing.New(ctx, `argoexec`)
		require.NoError(t, err)

		callOrder := []string{}
		mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "main", true).Run(func(args mock.Arguments) { callOrder = append(callOrder, "main") }).Return(io.NopCloser(strings.NewReader("main log")), nil)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "init", true).Run(func(args mock.Arguments) { callOrder = append(callOrder, "init") }).Return(io.NopCloser(strings.NewReader("init log")), nil)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "wait", true).Run(func(args mock.Arguments) { callOrder = append(callOrder, "wait") }).Return(io.NopCloser(strings.NewReader("wait log")), nil)

		we := WorkflowExecutor{
			Template:        templateWithMainAndSystemLogs,
			RuntimeExecutor: &mockRuntimeExecutor,
			Tracing:         tr,
		}

		we.SaveLogs(ctx)

		require.NotEmpty(t, callOrder)
		assert.Equal(t, "wait", callOrder[len(callOrder)-1], "wait logs should be saved last")
	})

	t.Run("init-less: supervisor archived, init and wait skipped", func(t *testing.T) {
		// In init-less pods there is no init/wait container - a single
		// supervisor container plays both roles. SaveLogs must archive the
		// supervisor's combined log and not touch init/wait.
		t.Setenv(common.EnvVarInitlessPod, "true")

		ctx := logging.TestContext(t.Context())
		tr, err := tracing.New(ctx, `argoexec`)
		require.NoError(t, err)

		mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "main", true).Return(io.NopCloser(strings.NewReader("main log")), nil)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "supervisor", true).Return(io.NopCloser(strings.NewReader("supervisor log")), nil)

		we := WorkflowExecutor{
			Template:        templateWithMainAndSystemLogs,
			RuntimeExecutor: &mockRuntimeExecutor,
			Tracing:         tr,
		}

		we.SaveLogs(ctx)

		mockRuntimeExecutor.AssertCalled(t, "GetOutputStream", mock.Anything, "main", true)
		mockRuntimeExecutor.AssertCalled(t, "GetOutputStream", mock.Anything, "supervisor", true)
		mockRuntimeExecutor.AssertNotCalled(t, "GetOutputStream", mock.Anything, "init", true)
		mockRuntimeExecutor.AssertNotCalled(t, "GetOutputStream", mock.Anything, "wait", true)
	})

	t.Run("init-less: supervisor skipped when combined file is missing", func(t *testing.T) {
		// ErrorNotExist must be skipped, not recorded as
		// an error - a returned error crash-loops the supervisor.
		t.Setenv(common.EnvVarInitlessPod, "true")

		ctx := logging.TestContext(t.Context())
		tr, err := tracing.New(ctx, `argoexec`)
		require.NoError(t, err)

		mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "main", true).Return(io.NopCloser(strings.NewReader("main log")), nil)
		mockRuntimeExecutor.On("GetOutputStream", mock.Anything, "supervisor", true).Return(nil, fs.ErrNotExist)

		we := WorkflowExecutor{
			Template:        templateWithMainAndSystemLogs,
			RuntimeExecutor: &mockRuntimeExecutor,
			Tracing:         tr,
		}

		we.SaveLogs(ctx)

		mockRuntimeExecutor.AssertCalled(t, "GetOutputStream", mock.Anything, "supervisor", true)
		// only main's storage error; supervisor is skipped on ErrNotExist.
		require.Len(t, we.errors, 1, "only the main container's storage error is expected")
		assert.EqualError(t, we.errors[0], artStorageError)
	})

	t.Run("container with no log file is skipped, not an error", func(t *testing.T) {
		// A container killed before its command ran (e.g. a containerSet member
		// whose dependency failed, then SIGTERM'd) has no combined log. SaveLogs
		// must skip it, not record an error — otherwise the init-less supervisor
		// crash-loops and the pod never completes.
		ctx := logging.TestContext(t.Context())
		tr, err := tracing.New(ctx, `argoexec`)
		require.NoError(t, err)
		rt := &mocks.ContainerRuntimeExecutor{}
		rt.On("GetOutputStream", mock.Anything, mock.AnythingOfType("string"), true).
			Return((io.ReadCloser)(nil), os.ErrNotExist)
		we := WorkflowExecutor{
			Template:        wfv1.Template{ArchiveLocation: &wfv1.ArtifactLocation{ArchiveLogs: new(true)}},
			RuntimeExecutor: rt,
			Tracing:         tr,
		}

		logArtifacts := we.SaveLogs(ctx)

		assert.Empty(t, logArtifacts)
		assert.NoError(t, we.HasError(), "a missing combined log must not be a fatal error")
	})
}

func TestReportOutputs(t *testing.T) {
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	mockTaskResultClient := argofake.NewClientset().ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace)
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
		ctx := logging.TestContext(t.Context())
		tracing, err := tracing.New(ctx, `argoexec`) // TODO arguments here
		require.NoError(t, err)
		we := WorkflowExecutor{
			Template:         templateWithArtifacts,
			RuntimeExecutor:  &mockRuntimeExecutor,
			taskResultClient: mockTaskResultClient,
			Tracing:          tracing,
		}

		err = we.ReportOutputs(ctx, artifacts)

		require.NoError(t, err)
		assert.Empty(t, we.errors)
	})
}

func TestUntarMaliciousSymlink(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a target directory outside the extraction root
	outsideDir := filepath.Join(tmpDir, "outside")
	err := os.Mkdir(outsideDir, 0755)
	require.NoError(t, err)

	// Create a file in the outside directory to verify it's NOT overwritten initially
	targetFile := filepath.Join(outsideDir, "pwned")
	err = os.WriteFile(targetFile, []byte("safe"), 0644)
	require.NoError(t, err)

	// Create the malicious tarball directly
	tarPath := filepath.Join(tmpDir, "malicious.tar.gz")
	f, err := os.Create(tarPath)
	require.NoError(t, err)
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// 1. Create a symlink "link" -> absolute path of outsideDir
	absOutside, err := filepath.Abs(outsideDir)
	require.NoError(t, err)

	err = tw.WriteHeader(&tar.Header{
		Name:     "link",
		Typeflag: tar.TypeSymlink,
		Linkname: absOutside,
		Mode:     0777,
	})
	require.NoError(t, err)

	// 2. Create a file "link/pwned" that writes through the symlink
	fileContent := []byte("pwned")
	err = tw.WriteHeader(&tar.Header{
		Name:     "link/pwned",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     int64(len(fileContent)),
	})
	require.NoError(t, err)
	_, err = tw.Write(fileContent)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	// Debug: List tarball contents
	cmd := exec.CommandContext(t.Context(), "tar", "-tvzf", tarPath)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	if err == nil {
		t.Logf("Tarball contents:\n%s", string(out))
	}

	// Destination directory for extraction
	destDir := filepath.Join(tmpDir, "dest")

	// Perform untar
	err = untar(tarPath, destDir)
	// This should return an error because the symlink is outside the extraction root
	require.Error(t, err)

	// Check if the file outside was overwritten
	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)

	// If content is "pwned", the vulnerability is reproduced.
	if string(content) == "pwned" {
		t.Logf("Tar slip symlink vulnerability reproduced: File outside was overwritten with '%s'", string(content))
	} else {
		t.Logf("Tar slip symlink vulnerability NOT reproduced: File content is '%s'", string(content))
	}

	// Assert that it IS "safe" (this should FAIL if vulnerable)
	assert.Equal(t, "safe", string(content), "File outside should NOT be overwritten")
}

func TestUnzipMaliciousSymlink(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a target directory outside the extraction root
	outsideDir := filepath.Join(tmpDir, "outside")
	err := os.Mkdir(outsideDir, 0755)
	require.NoError(t, err)

	// Create a file in the outside directory
	targetFile := filepath.Join(outsideDir, "pwned")
	err = os.WriteFile(targetFile, []byte("safe"), 0644)
	require.NoError(t, err)

	// Create the malicious zip
	zipPath := filepath.Join(tmpDir, "malicious.zip")
	f, err := os.Create(zipPath)
	require.NoError(t, err)
	zw := zip.NewWriter(f)

	// 1. Create a symlink "link" -> "../outside"
	header := &zip.FileHeader{
		Name:   "link",
		Method: zip.Store,
	}
	header.SetMode(0777 | os.ModeSymlink)
	w, err := zw.CreateHeader(header)
	require.NoError(t, err)
	_, err = w.Write([]byte("../outside"))
	require.NoError(t, err)

	// 2. Create a file "link/pwned"
	w, err = zw.Create("link/pwned")
	require.NoError(t, err)
	_, err = w.Write([]byte("pwned"))
	require.NoError(t, err)

	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())

	// Destination directory
	destDir := filepath.Join(tmpDir, "dest")

	// Perform unzip
	ctx := logging.TestContext(t.Context())
	// This should return an error because the symlink is outside the extraction root
	err = unzip(ctx, zipPath, destDir)
	require.Error(t, err)

	// Check if the file outside was overwritten
	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)

	assert.Equal(t, "safe", string(content), "File outside should NOT be overwritten by unzip")
}

func TestWaitMainContainers(t *testing.T) {
	t.Run("delegates to the runtime executor's exit-code file watch", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		rt := &mocks.ContainerRuntimeExecutor{}
		rt.On("Wait", mock.Anything, mock.Anything).Return(nil)
		we := &WorkflowExecutor{
			PodName:         fakePodName,
			Namespace:       fakeNamespace,
			RuntimeExecutor: rt,
		}
		require.NoError(t, we.waitMainContainers(ctx, []string{"main"}))
		rt.AssertExpectations(t)
	})
}
