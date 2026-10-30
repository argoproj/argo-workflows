package archive

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	workflowarchivepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestGetArchivedWorkflow(t *testing.T) {
	serviceClient := &mockArchivedWorkflowServiceClient{}
	expected := &wfv1.Workflow{}
	serviceClient.On("GetArchivedWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowarchivepkg.GetArchivedWorkflowRequest) bool {
		return req.Uid == "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" && req.Namespace == "example-workflows"
	}), mock.Anything).Return(expected, nil).Once()

	actual, err := getArchivedWorkflow(context.Background(), serviceClient, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", "example-workflows")

	require.NoError(t, err)
	assert.Same(t, expected, actual)
	serviceClient.AssertExpectations(t)
}
