package common

import (
	"bytes"
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

type MockKC struct {
	getContainerStatusPod             *v1.Pod
	getContainerStatusContainerStatus *v1.ContainerStatus
	getContainerStatusErr             error
	killContainerError                error
}

func (m *MockKC) GetContainerStatuses(ctx context.Context) (*v1.Pod, []v1.ContainerStatus, error) {
	return m.getContainerStatusPod, []v1.ContainerStatus{*m.getContainerStatusContainerStatus}, m.getContainerStatusErr
}

func (m *MockKC) GetContainerStatus(ctx context.Context, containerName string) (*v1.Pod, *v1.ContainerStatus, error) {
	return m.getContainerStatusPod, m.getContainerStatusContainerStatus, m.getContainerStatusErr
}

func (m *MockKC) KillContainer(pod *v1.Pod, container *v1.ContainerStatus, sig syscall.Signal) error {
	return m.killContainerError
}

func (*MockKC) CreateArchive(ctx context.Context, containerName, sourcePath string) (*bytes.Buffer, error) {
	return nil, nil
}

// TestWaitForTermination ensure we SIGTERM container with input wait time
func TestWaitForTermination(t *testing.T) {
	// Successfully SIGTERM Container
	mock := &MockKC{
		getContainerStatusContainerStatus: &v1.ContainerStatus{
			Name: "container-name",
			State: v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{},
			},
		},
	}
	ctx := context.Background()
	err := WaitForTermination(ctx, mock, []string{"container-name"}, time.Duration(10)*time.Second)
	assert.NoError(t, err)

	// Fail SIGTERM Container
	mock = &MockKC{
		getContainerStatusContainerStatus: &v1.ContainerStatus{
			Name: "container-name",
			State: v1.ContainerState{
				Terminated: nil,
			},
		},
	}
	err = WaitForTermination(ctx, mock, []string{"container-name"}, time.Duration(1)*time.Second)
	assert.EqualError(t, err, "timeout after 1s")
}
