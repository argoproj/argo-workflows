package executor

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/mocks"
)

const (
	fakePodName       = "fake-test-pod-1234567890"
	fakeWorkflow      = "my-wf"
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
			we := WorkflowExecutor{
				Template: wfv1.Template{
					Inputs: wfv1.Inputs{
						Artifacts: []wfv1.Artifact{test.artifact},
					},
				},
			}
			err := we.LoadArtifacts(context.Background())
			assert.EqualError(t, err, test.error)
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

	ctx := context.Background()
	err := we.SaveParameters(ctx)
	assert.NoError(t, err)
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

	ctx := context.Background()
	err := we.SaveParameters(ctx)
	assert.NoError(t, err)
	assert.Equal(t, we.Template.Outputs.Parameters[0].Value.String(), "Default Value")
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

	ctx := context.Background()
	err := we.SaveParameters(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "", we.Template.Outputs.Parameters[0].Value.String())
}

func TestIsTarball(t *testing.T) {
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
		ok, err := isTarball(test.path)
		if test.expectErr {
			assert.Error(t, err, test.path)
		} else {
			assert.NoError(t, err, test.path)
		}
		assert.Equal(t, test.isTarball, ok, test.path)
	}
}

func TestUnzip(t *testing.T) {
	zipPath := "testdata/file.zip"
	destPath := "testdata/unzippedFile"

	// test
	err := unzip(zipPath, destPath)
	assert.NoError(t, err)

	// check unzipped file
	fileInfo, err := os.Stat(destPath)
	assert.NoError(t, err)
	assert.True(t, fileInfo.Mode().IsRegular())

	// cleanup
	err = os.Remove(destPath)
	assert.NoError(t, err)
}

func TestUntar(t *testing.T) {
	tarPath := "testdata/file.tar.gz"
	destPath := "testdata/untarredFile"

	// test
	err := untar(tarPath, destPath)
	assert.NoError(t, err)

	// check untarred file
	fileInfo, err := os.Stat(destPath)
	assert.NoError(t, err)
	assert.True(t, fileInfo.Mode().IsRegular())

	// cleanup
	err = os.Remove(destPath)
	assert.NoError(t, err)
}

func TestChmod(t *testing.T) {
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

		tempFile, err := ioutil.TempFile(tempDir, "chmod-file-test")
		assert.NoError(t, err)

		// Run chmod function
		err = chmod(tempDir, test.mode, test.recurse)
		assert.NoError(t, err)

		// Check directory mode if set
		dirPermission, err := os.Stat(tempDir)
		assert.NoError(t, err)
		assert.Equal(t, dirPermission.Mode().String(), test.permissions.dir)

		// Check file mode mode if set
		filePermission, err := os.Stat(tempFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, filePermission.Mode().String(), test.permissions.file)
	}
}

func TestSaveArtifacts(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	templateWithOutParam := wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{
					Name: "samedir",
					Path: "/samedir",
				},
			},
		},
		Outputs: wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				{
					Name:     "samedir",
					Path:     "/samedir",
					Optional: true,
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

	ctx := context.Background()
	err := we.SaveArtifacts(ctx)
	assert.NoError(t, err)

	we.Template.Outputs.Artifacts[0].Optional = false
	err = we.SaveArtifacts(ctx)
	assert.Error(t, err)
}

func TestMonitorProgress(t *testing.T) {
	ctx := context.Background()

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
		nil,
		taskResults,
		nil,
		fakePodName,
		fakePodUID,
		fakeWorkflow,
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
	assert.NoError(t, err)

	time.Sleep(time.Second)

	result, err := taskResults.Get(ctx, fakeNodeID, metav1.GetOptions{})
	if assert.NoError(t, err) {
		assert.Equal(t, fakeWorkflow, result.Labels[common.LabelKeyWorkflow])
		assert.Len(t, result.OwnerReferences, 1)
		assert.Equal(t, wfv1.Progress("100/100"), result.Progress)
	}
}
