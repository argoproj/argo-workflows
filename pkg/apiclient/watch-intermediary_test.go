package apiclient

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
)

func TestWorkflowPodWatchIntermediarySend(t *testing.T) {
	t.Run("delivers the event to Recv", func(t *testing.T) {
		intermediary := newWorkflowPodWatchIntermediary(t.Context())
		event := &workflowpkg.WorkflowPodWatchEvent{Type: "ADDED"}
		sent := make(chan error, 1)
		go func() { sent <- intermediary.Send(event) }()

		received, err := intermediary.Recv()

		require.NoError(t, err)
		assert.Same(t, event, received)
		require.NoError(t, <-sent)
	})

	t.Run("returns the context error when cancelled with no receiver", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		intermediary := newWorkflowPodWatchIntermediary(ctx)
		sent := make(chan error, 1)
		go func() { sent <- intermediary.Send(&workflowpkg.WorkflowPodWatchEvent{}) }()

		cancel()

		select {
		case err := <-sent:
			require.ErrorIs(t, err, context.Canceled)
		case <-time.After(5 * time.Second):
			t.Fatal("Send did not return after the context was cancelled")
		}
	})
}
