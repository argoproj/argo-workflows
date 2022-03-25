package commands

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowmocks "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func Test_listWorkflows(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		workflows, err := listEmpty(&metav1.ListOptions{}, listFlags{})
		if assert.NoError(t, err) {
			assert.Len(t, workflows, 0)
		}
	})
	t.Run("Nothing", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{}, listFlags{})
		if assert.NoError(t, err) {
			assert.NotNil(t, workflows)
		}
	})
	t.Run("Status", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{LabelSelector: "workflows.argoproj.io/phase in (Pending,Running)"}, listFlags{status: []string{"Running", "Pending"}})
		if assert.NoError(t, err) {
			assert.NotNil(t, workflows)
		}
	})
	t.Run("Completed", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{LabelSelector: "workflows.argoproj.io/completed=true"}, listFlags{completed: true})
		if assert.NoError(t, err) {
			assert.NotNil(t, workflows)
		}
	})
	t.Run("Running", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{LabelSelector: "workflows.argoproj.io/completed!=true"}, listFlags{running: true})
		if assert.NoError(t, err) {
			assert.NotNil(t, workflows)
		}
	})
	t.Run("Resubmitted", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{LabelSelector: common.LabelKeyPreviousWorkflowName}, listFlags{resubmitted: true})
		if assert.NoError(t, err) {
			assert.NotNil(t, workflows)
		}
	})
	t.Run("Labels", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{LabelSelector: "foo"}, listFlags{labels: "foo"})
		if assert.NoError(t, err) {
			assert.NotNil(t, workflows)
		}
	})
	t.Run("Prefix", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{}, listFlags{prefix: "foo-"})
		if assert.NoError(t, err) {
			assert.Len(t, workflows, 1)
		}
	})
	t.Run("Since", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{}, listFlags{createdSince: "1h"})
		if assert.NoError(t, err) {
			assert.Len(t, workflows, 1)
		}
	})
	t.Run("Older", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{}, listFlags{finishedAfter: "1h"})
		if assert.NoError(t, err) {
			assert.Len(t, workflows, 1)
		}
	})
	t.Run("Names", func(t *testing.T) {
		workflows, err := list(&metav1.ListOptions{FieldSelector: nameFields}, listFlags{fields: nameFields})
		if assert.NoError(t, err) {
			assert.Len(t, workflows, 3)
			// most recent workflow will be shown first
			assert.Equal(t, "bar-", workflows[0].Name)
			assert.Equal(t, "baz-", workflows[1].Name)
			assert.Equal(t, "foo-", workflows[2].Name)
		}
	})
}

func list(listOptions *metav1.ListOptions, flags listFlags) (wfv1.Workflows, error) {
	c := &workflowmocks.WorkflowServiceClient{}
	c.On("ListWorkflows", mock.Anything, &workflow.WorkflowListRequest{ListOptions: listOptions, Fields: flags.displayFields()}).Return(&wfv1.WorkflowList{Items: wfv1.Workflows{
		{ObjectMeta: metav1.ObjectMeta{Name: "foo-", CreationTimestamp: metav1.Time{Time: time.Now().Add(-2 * time.Hour)}}, Status: wfv1.WorkflowStatus{FinishedAt: metav1.Time{Time: time.Now().Add(-2 * time.Hour)}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "bar-", CreationTimestamp: metav1.Time{Time: time.Now()}}},
		{ObjectMeta: metav1.ObjectMeta{
			Name:              "baz-",
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-2 * time.Hour)},
			Labels:            map[string]string{common.LabelKeyPreviousWorkflowName: "foo-"},
		}},
	}}, nil)
	workflows, err := listWorkflows(context.Background(), c, flags)
	return workflows, err
}

func listEmpty(listOptions *metav1.ListOptions, flags listFlags) (wfv1.Workflows, error) {
	c := &workflowmocks.WorkflowServiceClient{}
	c.On("ListWorkflows", mock.Anything, &workflow.WorkflowListRequest{ListOptions: listOptions, Fields: defaultFields}).Return(&wfv1.WorkflowList{Items: wfv1.Workflows{}}, nil)
	workflows, err := listWorkflows(context.Background(), c, flags)
	return workflows, err
}
