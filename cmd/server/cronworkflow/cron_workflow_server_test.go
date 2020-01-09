package cronworkflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/server/auth"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wftFake "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

func Test_cronWorkflowServiceServer(t *testing.T) {
	cronWf := &wfv1.CronWorkflow{
		ObjectMeta: v1.ObjectMeta{Namespace: "my-ns", Name: "my-name"},
	}
	wfClientset := wftFake.NewSimpleClientset(cronWf)
	server := NewCronWorkflowServer()
	ctx := context.WithValue(context.TODO(), auth.WfKey, wfClientset)

	// TODO - CreateCronWorkflow - which should replace the arg to NewSimpleClientset
	t.Run("ListCronWorkflows", func(t *testing.T) {
		cronWfs, err := server.ListCronWorkflows(ctx, &ListCronWorkflowsRequest{Namespace: "my-ns"})
		if assert.NoError(t, err) {
			assert.Len(t, cronWfs.Items, 1)
		}
	})
	t.Run("GetCronWorkflow", func(t *testing.T) {
		cronWf, err := server.GetCronWorkflow(ctx, &GetCronWorkflowRequest{Namespace: "my-ns", CronWorkflowName: "my-name"})
		if assert.NoError(t, err) {
			assert.NotNil(t, cronWf)
		}
	})
	t.Run("UpdateCronWorkflow", func(t *testing.T) {
		cronWf, err := server.UpdateCronWorkflow(ctx, &UpdateCronWorkflowRequest{Namespace: "my-ns", CronWorkflowName: "my-name", CronWorkflow: cronWf})
		if assert.NoError(t, err) {
			assert.NotNil(t, cronWf)
		}
	})
	t.Run("DeleteCronWorkflow", func(t *testing.T) {
		_, err := server.DeleteCronWorkflow(ctx, &DeleteCronWorkflowRequest{CronWorkflowName: "my-name", Namespace: "my-ns"})
		assert.NoError(t, err)
	})
}
