package cronworkflow

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/server/auth"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type cronWorkflowServiceServer struct {
}

func NewCronWorkflowServer() CronWorkflowServiceServer {
	return &cronWorkflowServiceServer{}
}

func (c *cronWorkflowServiceServer) ListWorkflowTemplates(ctx context.Context, req *ListCronWorkflowsRequest) (*v1alpha1.CronWorkflowList, error) {
	options := &metav1.ListOptions{}
	if req.ListOptions != nil {
		options = req.ListOptions
	}
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).List(*options)
}
