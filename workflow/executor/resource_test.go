package executor

import (
	"context"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/retry"

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
		PodName:         fakePodName,
		Template:        template,
		ClientSet:       fakeClientset,
		Namespace:       fakeNamespace,
		RuntimeExecutor: &mockRuntimeExecutor,
	}
	args, err := we.getKubectlArguments("fake", manifestPath, fakeFlags)
	assert.NoError(t, err)
	assert.Contains(t, args, fakeFlags[0])

	_, err = we.getKubectlArguments("fake", manifestPath, nil)
	assert.NoError(t, err)
	_, err = we.getKubectlArguments("fake", "unknown-location", fakeFlags)
	if runtime.GOOS == "windows" {
		assert.EqualError(t, err, "open unknown-location: The system cannot find the file specified.")
	} else {
		assert.EqualError(t, err, "open unknown-location: no such file or directory")
	}

	emptyFile, err := os.CreateTemp("/tmp", "empty-manifest")
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
	buff, err := os.ReadFile(manifestPath)
	assert.NoError(t, err)
	fakeFlags := []string{"kubectl", "patch", "--type", "strategic", "-p", string(buff), "-f", manifestPath, "-o", "json"}

	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}

	template := wfv1.Template{
		Resource: &wfv1.ResourceTemplate{
			Action: "patch",
			Flags:  fakeFlags,
		},
	}

	we := WorkflowExecutor{
		PodName:         fakePodName,
		Template:        template,
		ClientSet:       fakeClientset,
		Namespace:       fakeNamespace,
		RuntimeExecutor: &mockRuntimeExecutor,
	}
	args, err := we.getKubectlArguments("patch", manifestPath, nil)

	assert.NoError(t, err)
	assert.Equal(t, args, fakeFlags)
}

// TestResourcePatchFlagsJson tests whether Resource Flags
// are properly passed to `kubectl patch` command in json patches
func TestResourcePatchFlagsJson(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	manifestPath := "../../examples/hello-world.yaml"
	buff, err := os.ReadFile(manifestPath)
	assert.NoError(t, err)
	fakeFlags := []string{"kubectl", "patch", "--type", "json", "-p", string(buff), "-o", "json"}

	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}

	template := wfv1.Template{
		Resource: &wfv1.ResourceTemplate{
			Action:        "patch",
			Flags:         fakeFlags,
			MergeStrategy: "json",
		},
	}

	we := WorkflowExecutor{
		PodName:         fakePodName,
		Template:        template,
		ClientSet:       fakeClientset,
		Namespace:       fakeNamespace,
		RuntimeExecutor: &mockRuntimeExecutor,
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

// TestInferSelfLink tests whether the inferred self link for k8s objects are correct.
func TestInferSelfLink(t *testing.T) {
	obj := unstructured.Unstructured{}
	obj.SetNamespace("test-namespace")
	obj.SetName("test-name")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	})
	assert.Equal(t, "api/v1/namespaces/test-namespace/pods/test-name", inferObjectSelfLink(obj))

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "test.group",
		Version: "v1",
		Kind:    "TestKind",
	})
	assert.Equal(t, "apis/test.group/v1/namespaces/test-namespace/testkinds/test-name", inferObjectSelfLink(obj))

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "test.group",
		Version: "v1",
		Kind:    "Duty",
	})
	assert.Equal(t, "apis/test.group/v1/namespaces/test-namespace/duties/test-name", inferObjectSelfLink(obj))

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "test.group",
		Version: "v1",
		Kind:    "IngressGateway",
	})
	assert.Equal(t, "apis/test.group/v1/namespaces/test-namespace/ingressgateways/test-name", inferObjectSelfLink(obj))

	obj.SetNamespace("")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	})
	assert.Equal(t, "api/v1/namespaces/test-name", inferObjectSelfLink(obj))

}

// TestResourceExecRetry tests whether Exec retries transitive errors
func TestResourceExecRetry(t *testing.T) {
	we := WorkflowExecutor{
		PodName:         fakePodName,
		Template:        wfv1.Template{},
		ClientSet:       fake.NewSimpleClientset(),
		Namespace:       fakeNamespace,
		RuntimeExecutor: &mocks.ContainerRuntimeExecutor{},
	}

	_, filename, _, _ := runtime.Caller(0)
	dirname := path.Dir(filename)
	duration := retry.DefaultBackoff.Duration
	path := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", path)
		retry.DefaultBackoff.Duration = duration
	}()
	retry.DefaultBackoff.Duration = 0
	os.Setenv("PATH", dirname+"/testdata")

	_, _, _, err := we.ExecResource("", "../../examples/hello-world.yaml", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no more retries")
}

func Test_jqFilter(t *testing.T) {
	for _, testCase := range []struct {
		input  []byte
		filter string
		want   string
	}{
		{[]byte(`{"metadata": {"name": "foo"}}`), ".metadata.name", "foo"},
		{[]byte(`{"items": [{"key": "foo"}, {"key": "bar"}]}`), ".items.[].key", "foo\nbar"},
	} {
		t.Run(string(testCase.input), func(t *testing.T) {
			ctx := context.Background()
			got, err := jqFilter(ctx, testCase.input, testCase.filter)
			assert.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func Test_runKubectl(t *testing.T) {
	out, err := runKubectl("kubectl", "version", "--client=true", "--output", "json")
	assert.NoError(t, err)
	assert.Contains(t, string(out), "clientVersion")
}
