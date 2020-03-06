package cronworkflow

import (
	"context"
	"testing"

	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/server/auth"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wftFake "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

func Test_cronWorkflowServiceServer(t *testing.T) {
	cronWf := &wfv1.CronWorkflow{
		ObjectMeta: v1.ObjectMeta{Namespace: "my-ns", Name: "my-name"},
	}
	wfClientset := wftFake.NewSimpleClientset()
	server := NewCronWorkflowServer(GRPCServerMode, "")
	ctx := context.WithValue(context.TODO(), auth.WfKey, wfClientset)

	t.Run("CreateCronWorkflow", func(t *testing.T) {
		created, err := server.CreateCronWorkflow(ctx, &cronworkflowpkg.CreateCronWorkflowRequest{
			Namespace:    "my-ns",
			CronWorkflow: cronWf,
		})
		if assert.NoError(t, err) {
			assert.NotNil(t, created)
		}
	})
	t.Run("ListCronWorkflows", func(t *testing.T) {
		cronWfs, err := server.ListCronWorkflows(ctx, &cronworkflowpkg.ListCronWorkflowsRequest{Namespace: "my-ns"})
		if assert.NoError(t, err) {
			assert.Len(t, cronWfs.Items, 1)
		}
	})
	t.Run("GetCronWorkflow", func(t *testing.T) {
		cronWf, err := server.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{Namespace: "my-ns", Name: "my-name"})
		if assert.NoError(t, err) {
			assert.NotNil(t, cronWf)
		}
	})
	t.Run("UpdateCronWorkflow", func(t *testing.T) {
		cronWf, err := server.UpdateCronWorkflow(ctx, &cronworkflowpkg.UpdateCronWorkflowRequest{Namespace: "my-ns", Name: "my-name", CronWorkflow: cronWf})
		if assert.NoError(t, err) {
			assert.NotNil(t, cronWf)
		}
	})
	t.Run("DeleteCronWorkflow", func(t *testing.T) {
		_, err := server.DeleteCronWorkflow(ctx, &cronworkflowpkg.DeleteCronWorkflowRequest{Name: "my-name", Namespace: "my-ns"})
		assert.NoError(t, err)
	})
}
