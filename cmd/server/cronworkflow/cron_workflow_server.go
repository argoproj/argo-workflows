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

func (c *cronWorkflowServiceServer) ListCronWorkflows(ctx context.Context, req *ListCronWorkflowsRequest) (*v1alpha1.CronWorkflowList, error) {
	options := metav1.ListOptions{}
	if req.ListOptions != nil {
		options = *req.ListOptions
	}
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).List(options)
}

func (c *cronWorkflowServiceServer) CreateCronWorkflow(ctx context.Context, req *CreateCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Create(req.CronWorkflow)
}

func (c *cronWorkflowServiceServer) GetCronWorkflow(ctx context.Context, req *GetCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	options := metav1.GetOptions{}
	if req.GetOptions != nil {
		options = *req.GetOptions
	}
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Get(req.CronWorkflowName, options)
}

// TODO - UpdateCronWorkflow

func (c *cronWorkflowServiceServer) DeleteCronWorkflow(ctx context.Context, req *DeleteCronWorkflowRequest) (*CronWorkflowDeletedResponse, error) {
	err := auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Delete(req.CronWorkflowName, req.DeleteOptions)
	if err != nil {
		return nil, err
	}
	return &CronWorkflowDeletedResponse{}, nil
}
