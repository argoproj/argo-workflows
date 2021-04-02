package executor

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
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

// TestResourceConditionsMatching tests whether the JSON response match
// with either success or failure conditions.
func TestResourceConditionsMatching(t *testing.T) {
	var successReqs labels.Requirements
	successSelector, err := labels.Parse("status.phase == Succeeded")
	assert.NoError(t, err)
	successReqs, _ = successSelector.Requirements()
	assert.NoError(t, err)
	var failReqs labels.Requirements
	failSelector, err := labels.Parse("status.phase == Error")
	assert.NoError(t, err)
	failReqs, _ = failSelector.Requirements()
	assert.NoError(t, err)

	jsonBytes := []byte(`{"name": "test","status":{"phase":"Error"}`)
	finished, err := matchConditions(jsonBytes, successReqs, failReqs)
	assert.Error(t, err, `failure condition '{status.phase == [Error]}' evaluated true`)
	assert.False(t, finished)

	jsonBytes = []byte(`{"name": "test","status":{"phase":"Succeeded"}`)
	finished, err = matchConditions(jsonBytes, successReqs, failReqs)
	assert.NoError(t, err)
	assert.False(t, finished)

	jsonBytes = []byte(`{"name": "test","status":{"phase":"Pending"}`)
	finished, err = matchConditions(jsonBytes, successReqs, failReqs)
	assert.Error(t, err, "Neither success condition nor the failure condition has been matched. Retrying...")
	assert.True(t, finished)
}

// TestExecResource tests whether Resource is executed properly.
func TestExecResource(t *testing.T) {
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}
	template := wfv1.Template{
		Resource: &wfv1.ResourceTemplate{
			Action: "fake",
			Flags:  []string{"--fake=true"},
		},
	}
	we := WorkflowExecutor{
		PodName:            fakePodName,
		Template:           template,
		ClientSet:          fake.NewSimpleClientset(),
		Namespace:          fakeNamespace,
		PodAnnotationsPath: fakeAnnotations,
		ExecutionControl:   nil,
		RuntimeExecutor:    &mockRuntimeExecutor,
	}

	manifestWithoutGroup, err := ioutil.TempFile("/tmp", "manifest-without-group")
	assert.NoError(t, err)
	_, err = manifestWithoutGroup.WriteString(`apiVersion: ""
kind: Pod`)
	assert.NoError(t, err)
	defer func() { _ = os.Remove(manifestWithoutGroup.Name()) }()
	_, _, _, err = we.ExecResource("get", manifestWithoutGroup.Name(), nil)
	assert.EqualError(t, err, "Both group and name are required but at least one of them is missing from the manifest")

	manifestWithoutResName, err := ioutil.TempFile("/tmp", "manifest-without-resource-name")
	assert.NoError(t, err)
	_, err = manifestWithoutResName.WriteString(`apiVersion: v1
kind: Pod`)
	assert.NoError(t, err)
	defer func() { _ = os.Remove(manifestWithoutResName.Name()) }()
	_, _, _, err = we.ExecResource("get", manifestWithoutResName.Name(), nil)
	assert.EqualError(t, err, "Both group and name are required but at least one of them is missing from the manifest")

	manifestWithoutKind, err := ioutil.TempFile("/tmp", "manifest-without-resource-kind")
	assert.NoError(t, err)
	_, err = manifestWithoutKind.WriteString(`apiVersion: v1`)
	assert.NoError(t, err)
	defer func() { _ = os.Remove(manifestWithoutKind.Name()) }()
	_, _, _, err = we.ExecResource("get", manifestWithoutKind.Name(), nil)
	assert.NotNil(t, err)
	assert.Regexp(t, "error: unable to decode.*: Object 'Kind' is missing in.*", err.Error())
}
