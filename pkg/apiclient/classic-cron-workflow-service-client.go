package apiclient

import (
	"context"

	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
)

type classicCronWorkflowServiceClient struct {
	versioned.Interface
}

func (c *classicCronWorkflowServiceClient) CreateCronWorkflow(_ context.Context, _ *cronworkflow.CreateCronWorkflowRequest, opts ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	panic("implement me")
}

func (c *classicCronWorkflowServiceClient) ListCronWorkflows(_ context.Context, _ *cronworkflow.ListCronWorkflowsRequest, opts ...grpc.CallOption) (*v1alpha1.CronWorkflowList, error) {
	panic("implement me")
}

func (c *classicCronWorkflowServiceClient) GetCronWorkflow(_ context.Context, req *cronworkflow.GetCronWorkflowRequest, opts ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	options := metav1.GetOptions{}
	if req.GetOptions != nil {
		options = *req.GetOptions
	}
	return c.Interface.ArgoprojV1alpha1().CronWorkflows(req.GetNamespace()).Get(req.GetName(), options)
}

func (c *classicCronWorkflowServiceClient) UpdateCronWorkflow(_ context.Context, _ *cronworkflow.UpdateCronWorkflowRequest, opts ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	panic("implement me")
}

func (c *classicCronWorkflowServiceClient) DeleteCronWorkflow(_ context.Context, _ *cronworkflow.DeleteCronWorkflowRequest, opts ...grpc.CallOption) (*cronworkflow.CronWorkflowDeletedResponse, error) {
	panic("implement me")
}
