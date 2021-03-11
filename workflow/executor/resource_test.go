package executor

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/mocks"
)

// TestResourceFlags tests whether Resource Flags
// are properly passed to `kubectl` command
func TestResourceFlags(t *testing.T) {
	manifestPath := "../../examples/hello-world.yaml"
	fakeClientset := fake.NewSimpleClientset()
	fakeFlags := []string{"--fake=true"}

	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}

	template := wfv1.Template{
		Resource: &wfv1.ResourceTemplate{
			Action: "fake",
			Flags:  fakeFlags,
		},
	}

	we := WorkflowExecutor{
		PodName:            fakePodName,
		Template:           template,
		ClientSet:          fakeClientset,
		Namespace:          fakeNamespace,
		PodAnnotationsPath: fakeAnnotations,
		ExecutionControl:   nil,
		RuntimeExecutor:    &mockRuntimeExecutor,
	}
	args, err := we.getKubectlArguments("fake", manifestPath, fakeFlags)
	assert.NoError(t, err)
	assert.Contains(t, args, fakeFlags[0])

	_, err = we.getKubectlArguments("fake", manifestPath, nil)
	assert.NoError(t, err)
	_, err = we.getKubectlArguments("fake", "unknown-location", fakeFlags)
	assert.EqualError(t, err, "open unknown-location: no such file or directory")

	emptyFile, err := ioutil.TempFile("/tmp", "empty-manifest")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(emptyFile.Name()) }()
	_, err = we.getKubectlArguments("fake", emptyFile.Name(), nil)
	assert.EqualError(t, err, "Must provide at least one of flags or manifest.")
}

// TestResourcePatchFlags tests whether Resource Flags
// are properly passed to `kubectl patch` command
func TestResourcePatchFlags(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	manifestPath := "../../examples/hello-world.yaml"
	buff, err := ioutil.ReadFile(manifestPath)
	assert.NoError(t, err)
	fakeFlags := []string{"patch", "--type", "strategic", "-p", string(buff), "-o", "json"}

	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}

	template := wfv1.Template{
		Resource: &wfv1.ResourceTemplate{
			Action: "patch",
			Flags:  fakeFlags,
		},
	}

	we := WorkflowExecutor{
		PodName:            fakePodName,
		Template:           template,
		ClientSet:          fakeClientset,
		Namespace:          fakeNamespace,
		PodAnnotationsPath: fakeAnnotations,
		ExecutionControl:   nil,
		RuntimeExecutor:    &mockRuntimeExecutor,
	}
	args, err := we.getKubectlArguments("patch", manifestPath, nil)

	assert.NoError(t, err)
	assert.Equal(t, args, fakeFlags)
}
