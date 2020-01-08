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
	wfClientset := wftFake.NewSimpleClientset(&wfv1.CronWorkflow{
		ObjectMeta: v1.ObjectMeta{Namespace: "my-ns"},
	})
	server := NewCronWorkflowServer()
	ctx := context.WithValue(context.TODO(), auth.WfKey, wfClientset)

	t.Run("ListWorkflowTemplates", func(t *testing.T) {
		templates, err := server.ListWorkflowTemplates(ctx, &ListCronWorkflowsRequest{Namespace: "my-ns"})
		if assert.NoError(t, err) {
			assert.Len(t, templates.Items, 1)
		}
	})
}
