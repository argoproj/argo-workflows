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

func Test_retryWorkflows(t *testing.T) {
	t.Run("Retry workflow by names", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		retryOpts := retryOps{
			namespace: "argo",
		}
		cliSubmitOpts := common.CliSubmitOpts{}

		c.On("RetryWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := retryWorkflows(context.Background(), c, retryOpts, cliSubmitOpts, []string{"foo", "bar"})
		c.AssertNumberOfCalls(t, "RetryWorkflow", 2)

		assert.NoError(t, err)
	})

	t.Run("Retry workflow by selector", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		retryOpts := retryOps{
			namespace:     "argo",
			labelSelector: "custom-label=true",
		}
		cliSubmitOpts := common.CliSubmitOpts{}

		wfListReq := &workflowpkg.WorkflowListRequest{
			Namespace: "argo",
			ListOptions: &metav1.ListOptions{
				LabelSelector: retryOpts.labelSelector,
			},
			Fields: defaultFields,
		}

		wfList := &wfv1.WorkflowList{Items: wfv1.Workflows{
			{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "argo"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "bar", Namespace: "argo"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "baz", Namespace: "argo"}},
		}}

		c.On("ListWorkflows", mock.Anything, wfListReq).Return(wfList, nil)
		c.On("RetryWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := retryWorkflows(context.Background(), c, retryOpts, cliSubmitOpts, []string{})

		c.AssertNumberOfCalls(t, "RetryWorkflow", 3)
		for _, wf := range wfList.Items {
			retryReq := &workflowpkg.WorkflowRetryRequest{
				Name:              wf.Name,
				Namespace:         wf.Namespace,
				RestartSuccessful: retryOpts.restartSuccessful,
				NodeFieldSelector: "",
			}
			c.AssertCalled(t, "RetryWorkflow", mock.Anything, retryReq)
		}

		assert.NoError(t, err)
	})

	t.Run("Retry workflow by selector and name", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		retryOpts := retryOps{
			namespace:     "argo",
			labelSelector: "custom-label=true",
		}
		cliSubmitOpts := common.CliSubmitOpts{}

		wfListReq := &workflowpkg.WorkflowListRequest{
			Namespace: "argo",
			ListOptions: &metav1.ListOptions{
				LabelSelector: retryOpts.labelSelector,
			},
			Fields: defaultFields,
		}

		wfList := &wfv1.WorkflowList{Items: wfv1.Workflows{
			{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "bar"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "baz"}},
		}}

		c.On("ListWorkflows", mock.Anything, wfListReq).Return(wfList, nil)

		c.On("RetryWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)

		err := retryWorkflows(context.Background(), c, retryOpts, cliSubmitOpts, []string{"foo", "qux"})
		// after de-duplication, there will be 4 workflows to retry
		c.AssertNumberOfCalls(t, "RetryWorkflow", 4)

		// the 3 workflows from the selectors: "foo", "bar", "baz"
		for _, wf := range wfList.Items {
			retryReq := &workflowpkg.WorkflowRetryRequest{
				Name:              wf.Name,
				Namespace:         wf.Namespace,
				RestartSuccessful: retryOpts.restartSuccessful,
				NodeFieldSelector: "",
			}
			c.AssertCalled(t, "RetryWorkflow", mock.Anything, retryReq)
		}

		// the 1 workflow by the given name "qux
		c.AssertCalled(t, "RetryWorkflow", mock.Anything, &workflowpkg.WorkflowRetryRequest{
			Name:              "qux",
			Namespace:         "argo",
			RestartSuccessful: retryOpts.restartSuccessful,
			NodeFieldSelector: "",
		})

		assert.NoError(t, err)
	})

	t.Run("Retry workflow list error", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		retryOpts := retryOps{
			namespace:     "argo",
			labelSelector: "custom-label=true",
		}
		cliSubmitOpts := common.CliSubmitOpts{}
		c.On("ListWorkflows", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("mock error"))
		err := retryWorkflows(context.Background(), c, retryOpts, cliSubmitOpts, []string{})
		assert.Errorf(t, err, "mock error")
	})

	t.Run("Retry workflow error", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		retryOpts := retryOps{
			namespace: "argo",
		}
		cliSubmitOpts := common.CliSubmitOpts{}
		c.On("RetryWorkflow", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("mock error"))
		err := retryWorkflows(context.Background(), c, retryOpts, cliSubmitOpts, []string{"foo"})
		assert.Errorf(t, err, "mock error")
	})
}
