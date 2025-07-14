package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowmocks "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func Test_listWorkflows(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		workflows, err := listEmpty(t, &metav1.ListOptions{}, listFlags{})
		require.NoError(t, err)
		assert.Empty(t, workflows)
	})
	t.Run("Nothing", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{}, listFlags{})
		require.NoError(t, err)
		assert.NotNil(t, workflows)
	})
	t.Run("Status", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{LabelSelector: "workflows.argoproj.io/phase in (Pending,Running)"}, listFlags{status: []string{"Running", "Pending"}})
		require.NoError(t, err)
		assert.NotNil(t, workflows)
	})
	t.Run("Completed", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{LabelSelector: "workflows.argoproj.io/completed=true"}, listFlags{completed: true})
		require.NoError(t, err)
		assert.NotNil(t, workflows)
	})
	t.Run("Running", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{LabelSelector: "workflows.argoproj.io/completed!=true"}, listFlags{running: true})
		require.NoError(t, err)
		assert.NotNil(t, workflows)
	})
	t.Run("Resubmitted", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{LabelSelector: common.LabelKeyPreviousWorkflowName}, listFlags{resubmitted: true})
		require.NoError(t, err)
		assert.NotNil(t, workflows)
	})
	t.Run("Labels", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{LabelSelector: "foo"}, listFlags{labels: "foo"})
		require.NoError(t, err)
		assert.NotNil(t, workflows)
	})
	t.Run("Prefix", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{}, listFlags{prefix: "foo-"})
		require.NoError(t, err)
		assert.Len(t, workflows, 1)
	})
	t.Run("Since", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{}, listFlags{createdSince: "1h"})
		require.NoError(t, err)
		assert.Len(t, workflows, 1)
	})
	t.Run("Older", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{}, listFlags{finishedBefore: "1h"})
		require.NoError(t, err)
		assert.Len(t, workflows, 1)
	})
	t.Run("Names", func(t *testing.T) {
		workflows, err := list(t, &metav1.ListOptions{FieldSelector: nameFields}, listFlags{fields: nameFields})
		require.NoError(t, err)
		assert.Len(t, workflows, 3)
		// most recent workflow will be shown first
		assert.Equal(t, "bar-", workflows[0].Name)
		assert.Equal(t, "baz-", workflows[1].Name)
		assert.Equal(t, "foo-", workflows[2].Name)
	})
}

func list(t *testing.T, listOptions *metav1.ListOptions, flags listFlags) (wfv1.Workflows, error) {
	t.Helper()
	c := &workflowmocks.WorkflowServiceClient{}
	c.On("ListWorkflows", mock.Anything, &workflow.WorkflowListRequest{ListOptions: listOptions, Fields: flags.displayFields()}).Return(&wfv1.WorkflowList{Items: wfv1.Workflows{
		wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "foo-", CreationTimestamp: metav1.Time{Time: time.Now().Add(-2 * time.Hour)}}, Status: wfv1.WorkflowStatus{FinishedAt: metav1.Time{Time: time.Now().Add(-2 * time.Hour)}}},
		wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "bar-", CreationTimestamp: metav1.Time{Time: time.Now()}}},
		wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{
			Name:              "baz-",
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-2 * time.Hour)},
			Labels:            map[string]string{common.LabelKeyPreviousWorkflowName: "foo-"},
		}},
	}}, nil)
	ctx := logging.TestContext(t.Context())
	workflows, err := listWorkflows(ctx, c, flags)
	return workflows, err
}

func listEmpty(t *testing.T, listOptions *metav1.ListOptions, flags listFlags) (wfv1.Workflows, error) {
	t.Helper()
	c := &workflowmocks.WorkflowServiceClient{}
	c.On("ListWorkflows", mock.Anything, &workflow.WorkflowListRequest{ListOptions: listOptions, Fields: defaultFields}).Return(&wfv1.WorkflowList{Items: wfv1.Workflows{}}, nil)
	ctx := logging.TestContext(t.Context())
	workflows, err := listWorkflows(ctx, c, flags)
	return workflows, err
}
