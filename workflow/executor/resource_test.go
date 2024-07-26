package executor

import (
	"context"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	assert.Contains(t, args, fakeFlags[0])

	_, err = we.getKubectlArguments("fake", manifestPath, nil)
	require.NoError(t, err)
	_, err = we.getKubectlArguments("fake", "unknown-location", fakeFlags)
	if runtime.GOOS == "windows" {
		require.EqualError(t, err, "open unknown-location: The system cannot find the file specified.")
	} else {
		require.EqualError(t, err, "open unknown-location: no such file or directory")
	}

	emptyFile, err := os.CreateTemp("/tmp", "empty-manifest")
	require.NoError(t, err)
	defer func() { _ = os.Remove(emptyFile.Name()) }()
	_, err = we.getKubectlArguments("fake", emptyFile.Name(), nil)
	require.EqualError(t, err, "Must provide at least one of flags or manifest.")
}

// TestResourcePatchFlags tests whether Resource Flags
// are properly passed to `kubectl patch` command
func TestResourcePatchFlags(t *testing.T) {
	fakeFlags := []string{"pod", "mypod"}
	fakeClientset := fake.NewSimpleClientset()
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}

	tests := []struct {
		name           string
		patchType      string
		appendFileFlag bool
		manifestPath   string
	}{
		{
			name:           "strategic -f --patch-file",
			patchType:      "strategic",
			appendFileFlag: true,
			manifestPath:   "../../examples/hello-world.yaml", // any YAML with a `kind`
		},
		{
			name:           "json --patch-file",
			patchType:      "json",
			appendFileFlag: false,
			manifestPath:   "../../.golangci.yml", // any YAML without a `kind`
		},
		{
			name:           "merge --patch-file",
			patchType:      "merge",
			appendFileFlag: false,
			manifestPath:   "../../.golangci.yml", // any YAML without a `kind`
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedArgs := []string{"kubectl", "patch", "--type", tt.patchType, "--patch-file", tt.manifestPath}
			expectedArgs = append(expectedArgs, fakeFlags...)
			if tt.appendFileFlag {
				expectedArgs = append(expectedArgs, "-f", tt.manifestPath)
			}
			expectedArgs = append(expectedArgs, "-o", "json")

			template := wfv1.Template{
				Resource: &wfv1.ResourceTemplate{
					Action:        "patch",
					Flags:         fakeFlags,
					MergeStrategy: tt.patchType,
				},
			}
			we := WorkflowExecutor{
				PodName:         fakePodName,
				Template:        template,
				ClientSet:       fakeClientset,
				Namespace:       fakeNamespace,
				RuntimeExecutor: &mockRuntimeExecutor,
			}
			args, err := we.getKubectlArguments("patch", tt.manifestPath, fakeFlags)

			require.NoError(t, err)
			assert.Equal(t, expectedArgs, args)
		})
	}
}

// TestResourceConditionsMatching tests whether the JSON response match
// with either success or failure conditions.
func TestResourceConditionsMatching(t *testing.T) {
	var successReqs labels.Requirements
	successSelector, err := labels.Parse("status.phase == Succeeded")
	require.NoError(t, err)
	successReqs, _ = successSelector.Requirements()
	require.NoError(t, err)
	var failReqs labels.Requirements
	failSelector, err := labels.Parse("status.phase == Error")
	require.NoError(t, err)
	failReqs, _ = failSelector.Requirements()
	require.NoError(t, err)

	jsonBytes := []byte(`{"name": "test","status":{"phase":"Error"}`)
	finished, err := matchConditions(jsonBytes, successReqs, failReqs)
	require.Error(t, err, `failure condition '{status.phase == [Error]}' evaluated true`)
	assert.False(t, finished)

	jsonBytes = []byte(`{"name": "test","status":{"phase":"Succeeded"}`)
	finished, err = matchConditions(jsonBytes, successReqs, failReqs)
	require.NoError(t, err)
	assert.False(t, finished)

	jsonBytes = []byte(`{"name": "test","status":{"phase":"Pending"}`)
	finished, err = matchConditions(jsonBytes, successReqs, failReqs)
	require.Error(t, err, "Neither success condition nor the failure condition has been matched. Retrying...")
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
	defer func() {
		retry.DefaultBackoff.Duration = duration
	}()
	retry.DefaultBackoff.Duration = 0
	t.Setenv("PATH", dirname+"/testdata")

	_, _, _, err := we.ExecResource("", "../../examples/hello-world.yaml", nil)
	require.ErrorContains(t, err, "no more retries")
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
			require.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func Test_runKubectl(t *testing.T) {
	out, err := runKubectl("kubectl", "version", "--client=true", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, string(out), "clientVersion")
}
