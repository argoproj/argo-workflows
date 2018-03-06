package executor

import (
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/executor/mocks"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	fakePodName     = "fake-test-pod-1234567890"
	fakeNamespace   = "default"
	fakeAnnotations = "/tmp/podannotationspath"
	fakeContainerID = "abc123"
)

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
		PodName:            fakePodName,
		Template:           templateWithOutParam,
		ClientSet:          fakeClientset,
		Namespace:          fakeNamespace,
		PodAnnotationsPath: fakeAnnotations,
		ExecutionControl:   nil,
		RuntimeExecutor:    &mockRuntimeExecutor,
		mainContainerID:    fakeContainerID,
	}
	mockRuntimeExecutor.On("GetFileContents", fakeContainerID, "/path").Return("has a newline\n", nil)
	err := we.SaveParameters()
	assert.Nil(t, err)
	assert.Equal(t, *we.Template.Outputs.Parameters[0].Value, "has a newline")
}
