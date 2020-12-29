package common

import (
	"bytes"
	"context"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type MockKC struct {
	getContainerStatusPod             *v1.Pod
	getContainerStatusContainerStatus *v1.ContainerStatus
	getContainerStatusErr             error
	killContainerError                error
}

func (m *MockKC) GetContainerStatus(ctx context.Context, containerID string) (*v1.Pod, *v1.ContainerStatus, error) {
	return m.getContainerStatusPod, m.getContainerStatusContainerStatus, m.getContainerStatusErr
}

func (m *MockKC) KillContainer(pod *v1.Pod, container *v1.ContainerStatus, sig syscall.Signal) error {
	return m.killContainerError
}

func (*MockKC) CreateArchive(ctx context.Context, containerID, sourcePath string) (*bytes.Buffer, error) {
	return nil, nil
}

// TestScriptTemplateWithVolume ensure we can a script pod with input artifacts
func TestTerminatePodWithContainerID(t *testing.T) {
	// Already terminated.
	mock := &MockKC{
		getContainerStatusContainerStatus: &v1.ContainerStatus{
			State: v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{},
			},
		},
	}
	ctx := context.Background()
	err := TerminatePodWithContainerID(ctx, mock, "container-id", syscall.SIGTERM)
	assert.NoError(t, err)

	// w/ ShareProcessNamespace.
	mock = &MockKC{
		getContainerStatusPod: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: v1.PodSpec{
				ShareProcessNamespace: pointer.BoolPtr(true),
			},
		},
		getContainerStatusContainerStatus: &v1.ContainerStatus{
			State: v1.ContainerState{
				Terminated: nil,
			},
		},
	}
	err = TerminatePodWithContainerID(ctx, mock, "container-id", syscall.SIGTERM)
	assert.EqualError(t, err, "cannot terminate a process-namespace-shared Pod foo")

	// w/ HostPID.
	mock = &MockKC{
		getContainerStatusPod: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: v1.PodSpec{
				HostPID: true,
			},
		},
		getContainerStatusContainerStatus: &v1.ContainerStatus{
			State: v1.ContainerState{
				Terminated: nil,
			},
		},
	}
	err = TerminatePodWithContainerID(ctx, mock, "container-id", syscall.SIGTERM)
	assert.EqualError(t, err, "cannot terminate a hostPID Pod foo")

	// w/ RestartPolicy.
	mock = &MockKC{
		getContainerStatusPod: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: v1.PodSpec{
				RestartPolicy: "Always",
			},
		},
		getContainerStatusContainerStatus: &v1.ContainerStatus{
			State: v1.ContainerState{
				Terminated: nil,
			},
		},
	}
	err = TerminatePodWithContainerID(ctx, mock, "container-id", syscall.SIGTERM)
	assert.EqualError(t, err, "cannot terminate pod with a \"Always\" restart policy")

	// Successfully call KillContainer of the client interface.
	mock = &MockKC{
		getContainerStatusPod: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: v1.PodSpec{
				RestartPolicy: "Never",
			},
		},
		getContainerStatusContainerStatus: &v1.ContainerStatus{
			State: v1.ContainerState{
				Terminated: nil,
			},
		},
	}
	err = TerminatePodWithContainerID(ctx, mock, "container-id", syscall.SIGTERM)
	assert.NoError(t, err)
}
