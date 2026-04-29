package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	workflowmocks "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func Test_submitWorkflows(t *testing.T) {
	t.Run("Submit workflow with invalid options", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		ctx := logging.TestContext(t.Context())
		err := submitWorkflows(ctx, c, "argo", []wfv1.Workflow{}, &wfv1.SubmitOpts{}, &common.CliSubmitOpts{Watch: true, Wait: true})
		require.Error(t, err, "--wait cannot be combined with --watch")
	})
	t.Run("Submit without providing workflow", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		ctx := logging.TestContext(t.Context())
		err := submitWorkflows(ctx, c, "argo", []wfv1.Workflow{}, &wfv1.SubmitOpts{}, &common.CliSubmitOpts{})
		require.Error(t, err, "No Workflow found in given files")
	})
	t.Run("Submit workflow with priority set in spec", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		priority := int32(70)
		workflow := wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "argo"}, Spec: wfv1.WorkflowSpec{Priority: &priority}}

		c.On("CreateWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)
		ctx := logging.TestContext(t.Context())
		err := submitWorkflows(ctx, c, "argo", []wfv1.Workflow{workflow}, &wfv1.SubmitOpts{}, &common.CliSubmitOpts{})

		require.NoError(t, err)
		arg := c.Mock.Calls[0].Arguments[1]
		wfC, ok := arg.(*workflowpkg.WorkflowCreateRequest)
		if !ok {
			assert.Fail(t, "type is not WorkflowCreateRequest")
		}
		assert.Equal(t, priority, *wfC.Workflow.Spec.Priority)
	})

	t.Run("Submit workflow with priority set from cli", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		priority := int32(70)
		workflow := wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "argo"}, Spec: wfv1.WorkflowSpec{Priority: &priority}}

		priorityCLI := int32(100)
		cliSubmitOpts := common.CliSubmitOpts{Priority: &priorityCLI}

		c.On("CreateWorkflow", mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)
		ctx := logging.TestContext(t.Context())
		err := submitWorkflows(ctx, c, "argo", []wfv1.Workflow{workflow}, &wfv1.SubmitOpts{}, &cliSubmitOpts)

		require.NoError(t, err)
		arg := c.Mock.Calls[0].Arguments[1]
		wfC, ok := arg.(*workflowpkg.WorkflowCreateRequest)
		if !ok {
			assert.Fail(t, "type is not WorkflowCreateRequest")
		}
		assert.Equal(t, priorityCLI, *wfC.Workflow.Spec.Priority)
	})
}
