package http1

import (
	"context"

	"google.golang.org/grpc"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type ClusterWorkflowTemplateServiceClient = Facade

func (h ClusterWorkflowTemplateServiceClient) CreateClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/cluster-workflow-templates")
}

func (h ClusterWorkflowTemplateServiceClient) GetClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Get(in, out, "/api/v1/cluster-workflow-templates/{name}")
}

func (h ClusterWorkflowTemplateServiceClient) ListClusterWorkflowTemplates(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateListRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplateList, error) {
	out := &wfv1.ClusterWorkflowTemplateList{}
	return out, h.Get(in, out, "/api/v1/cluster-workflow-templates")
}

func (h ClusterWorkflowTemplateServiceClient) UpdateClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/cluster-workflow-templates/{name}")
}
