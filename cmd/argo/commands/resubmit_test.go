package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowmocks "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func Test_resubmitWorkflows(t *testing.T) {
	t.Run("Resubmit workflow by names", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		resubmitOpts := resubmitOps{
			namespace: "argo",
		}
		cliSubmitOpts := common.CliSubmitOpts{}

		c.On("ResubmitWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := resubmitWorkflows(context.Background(), c, resubmitOpts, cliSubmitOpts, []string{"foo", "bar"})
		c.AssertNumberOfCalls(t, "ResubmitWorkflow", 2)

		assert.NoError(t, err)
	})

	t.Run("Resubmit workflow with memoization", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		resubmitOpts := resubmitOps{
			namespace: "argo",
			memoized:  true,
		}
		cliSubmitOpts := common.CliSubmitOpts{}

		c.On("ResubmitWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := resubmitWorkflows(context.Background(), c, resubmitOpts, cliSubmitOpts, []string{"foo"})
		c.AssertNumberOfCalls(t, "ResubmitWorkflow", 1)
		c.AssertCalled(t, "ResubmitWorkflow", mock.Anything, &workflowpkg.WorkflowResubmitRequest{
			Name:      "foo",
			Namespace: "argo",
			Memoized:  true,
		})

		assert.NoError(t, err)
	})

	t.Run("Resubmit workflow by selector", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		resubmitOpts := resubmitOps{
			namespace:     "argo",
			labelSelector: "custom-label=true",
		}
		cliSubmitOpts := common.CliSubmitOpts{}

		wfListReq := &workflowpkg.WorkflowListRequest{
			Namespace: "argo",
			ListOptions: &metav1.ListOptions{
				LabelSelector: resubmitOpts.labelSelector,
			},
			Fields: defaultFields,
		}

		wfList := &wfv1.WorkflowList{Items: wfv1.Workflows{
			{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "argo"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "bar", Namespace: "argo"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "baz", Namespace: "argo"}},
		}}

		c.On("ListWorkflows", mock.Anything, wfListReq).Return(wfList, nil)
		c.On("ResubmitWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := resubmitWorkflows(context.Background(), c, resubmitOpts, cliSubmitOpts, []string{})

		c.AssertNumberOfCalls(t, "ResubmitWorkflow", 3)
		for _, wf := range wfList.Items {
			resubmitReq := &workflowpkg.WorkflowResubmitRequest{
				Name:      wf.Name,
				Namespace: wf.Namespace,
				Memoized:  false,
			}
			c.AssertCalled(t, "ResubmitWorkflow", mock.Anything, resubmitReq)
		}

		assert.NoError(t, err)
	})

	t.Run("Resubmit workflow by selector and name", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		resubmitOpts := resubmitOps{
			namespace:     "argo",
			labelSelector: "custom-label=true",
		}
		cliSubmitOpts := common.CliSubmitOpts{}

		wfListReq := &workflowpkg.WorkflowListRequest{
			Namespace: "argo",
			ListOptions: &metav1.ListOptions{
				LabelSelector: resubmitOpts.labelSelector,
			},
			Fields: defaultFields,
		}

		wfList := &wfv1.WorkflowList{Items: wfv1.Workflows{
			{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "bar"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "baz"}},
		}}

		c.On("ListWorkflows", mock.Anything, wfListReq).Return(wfList, nil)

		c.On("ResubmitWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := resubmitWorkflows(context.Background(), c, resubmitOpts, cliSubmitOpts, []string{"foo", "qux"})
		// after de-duplication, there will be 4 workflows to resubmit
		c.AssertNumberOfCalls(t, "ResubmitWorkflow", 4)

		// the 3 workflows from the selectors: "foo", "bar", "baz"
		for _, wf := range wfList.Items {
			resubmitReq := &workflowpkg.WorkflowResubmitRequest{
				Name:      wf.Name,
				Namespace: wf.Namespace,
				Memoized:  false,
			}
			c.AssertCalled(t, "ResubmitWorkflow", mock.Anything, resubmitReq)
		}

		// the 1 workflow by the given name "qux
		c.AssertCalled(t, "ResubmitWorkflow", mock.Anything, &workflowpkg.WorkflowResubmitRequest{
			Name:      "qux",
			Namespace: "argo",
			Memoized:  false,
		})

		assert.NoError(t, err)
	})

	t.Run("Resubmit workflow list error", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		resubmitOpts := resubmitOps{
			namespace:     "argo",
			labelSelector: "custom-label=true",
		}
		cliSubmitOpts := common.CliSubmitOpts{}
		c.On("ListWorkflows", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("mock error"))
		err := resubmitWorkflows(context.Background(), c, resubmitOpts, cliSubmitOpts, []string{})
		assert.Errorf(t, err, "mock error")
	})

	t.Run("Resubmit workflow error", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		resubmitOpts := resubmitOps{
			namespace: "argo",
		}
		cliSubmitOpts := common.CliSubmitOpts{}
		c.On("ResubmitWorkflow", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("mock error"))
		err := resubmitWorkflows(context.Background(), c, resubmitOpts, cliSubmitOpts, []string{"foo"})
		assert.Errorf(t, err, "mock error")
	})
}
