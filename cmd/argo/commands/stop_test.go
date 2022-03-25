package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowmocks "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func Test_stopWorkflows(t *testing.T) {
	t.Run("Stop workflow dry-run", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		stopArgs := stopOps{
			dryRun: true,
		}

		err := stopWorkflows(context.Background(), c, stopArgs, []string{"foo", "bar"})
		c.AssertNotCalled(t, "StopWorkflow")

		assert.NoError(t, err)
	})

	t.Run("Stop workflow by names", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		stopArgs := stopOps{
			namespace: "argo",
		}

		c.On("StopWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := stopWorkflows(context.Background(), c, stopArgs, []string{"foo", "bar"})
		c.AssertNumberOfCalls(t, "StopWorkflow", 2)

		assert.NoError(t, err)
	})

	t.Run("Stop workflow by selector", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		stopArgs := stopOps{
			namespace:     "argo",
			labelSelector: "custom-label=true",
		}

		wfListReq := &workflowpkg.WorkflowListRequest{
			Namespace: "argo",
			ListOptions: &metav1.ListOptions{
				LabelSelector: stopArgs.labelSelector,
			},
			Fields: defaultFields,
		}

		c.On("ListWorkflows", mock.Anything, wfListReq).Return(&wfv1.WorkflowList{Items: wfv1.Workflows{
			{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "bar"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "baz"}},
		}}, nil)

		c.On("StopWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := stopWorkflows(context.Background(), c, stopArgs, []string{})
		c.AssertNumberOfCalls(t, "StopWorkflow", 3)

		assert.NoError(t, err)
	})

	t.Run("Stop workflow by selector and name", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		stopArgs := stopOps{
			namespace:     "argo",
			labelSelector: "custom-label=true",
		}

		wfListReq := &workflowpkg.WorkflowListRequest{
			Namespace: "argo",
			ListOptions: &metav1.ListOptions{
				LabelSelector: stopArgs.labelSelector,
			},
			Fields: defaultFields,
		}

		c.On("ListWorkflows", mock.Anything, wfListReq).Return(&wfv1.WorkflowList{Items: wfv1.Workflows{
			{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "bar"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "baz"}},
		}}, nil)

		c.On("StopWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := stopWorkflows(context.Background(), c, stopArgs, []string{"foo", "qux"})
		// after de-duplication, there will be 4 workflows to stop
		c.AssertNumberOfCalls(t, "StopWorkflow", 4)

		assert.NoError(t, err)
	})

	t.Run("Stop workflow list error", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		stopArgs := stopOps{
			namespace:     "argo",
			labelSelector: "custom-label=true",
		}
		c.On("ListWorkflows", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("mock error"))
		err := stopWorkflows(context.Background(), c, stopArgs, []string{})
		assert.Errorf(t, err, "mock error")
	})

	t.Run("Stop workflow error", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		stopArgs := stopOps{
			namespace: "argo",
		}
		c.On("StopWorkflow", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("mock error"))
		err := stopWorkflows(context.Background(), c, stopArgs, []string{"foo"})
		assert.Errorf(t, err, "mock error")
	})
}
