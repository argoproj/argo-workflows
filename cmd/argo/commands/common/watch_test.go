package common

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfmocks "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow/mocks"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// mockWatchStream implements WorkflowService_WatchWorkflowsClient
type mockWatchStream struct {
	grpc.ClientStream
	recvCh      chan *workflowpkg.WorkflowWatchEvent
	recvStarted chan struct{}
}

func (m *mockWatchStream) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	// Signal that Recv has started blocking, then block until an event is
	// available; never return an error. The watch loop exits via ctx.Done(),
	// not via Recv errors.
	close(m.recvStarted)
	return <-m.recvCh, nil
}

func (m *mockWatchStream) Header() (metadata.MD, error) { return nil, nil }
func (m *mockWatchStream) Trailer() metadata.MD         { return nil }
func (m *mockWatchStream) CloseSend() error             { return nil }
func (m *mockWatchStream) Context() context.Context     { return context.Background() }
func (m *mockWatchStream) SendMsg(any) error            { return nil }
func (m *mockWatchStream) RecvMsg(any) error            { return nil }

func TestWatchWorkflow_ContextCancellation(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.Info, logging.Text))
	ctx, cancel := context.WithCancel(ctx)

	stream := &mockWatchStream{
		recvCh:      make(chan *workflowpkg.WorkflowWatchEvent),
		recvStarted: make(chan struct{}),
	}

	mockClient := wfmocks.NewWorkflowServiceClient(t)
	mockClient.On("WatchWorkflows", mock.Anything, mock.Anything).Return(stream, nil)

	done := make(chan error, 1)
	go func() {
		done <- WatchWorkflow(ctx, mockClient, "default", "test-wf", GetFlags{})
	}()

	// Wait until the watch stream is actually blocking on Recv, then cancel the
	// context to simulate Ctrl+C. This deterministically exercises cancellation
	// while the watch is in-flight.
	select {
	case <-stream.recvStarted:
		cancel()
	case <-time.After(5 * time.Second):
		t.Fatal("WatchWorkflow did not start the watch stream")
	}

	select {
	case err := <-done:
		require.NoError(t, err, "WatchWorkflow should return nil on context cancellation")
	case <-time.After(5 * time.Second):
		t.Fatal("WatchWorkflow did not exit after context cancellation - Ctrl+C would hang")
	}
}
