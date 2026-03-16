package apiclient

import (
	"context"

	"google.golang.org/grpc"

	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type argoKubeWorkflowClusterTemplateServiceClient struct {
	delegate clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceServer
}

var _ clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient = &argoKubeWorkflowClusterTemplateServiceClient{}

func (a *argoKubeWorkflowClusterTemplateServiceClient) CreateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateCreateRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	return a.delegate.CreateClusterWorkflowTemplate(ctx, req)
}

func (a *argoKubeWorkflowClusterTemplateServiceClient) GetClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	return a.delegate.GetClusterWorkflowTemplate(ctx, req)
}

func (a *argoKubeWorkflowClusterTemplateServiceClient) ListClusterWorkflowTemplates(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateListRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplateList, error) {
	return a.delegate.ListClusterWorkflowTemplates(ctx, req)
}

func (a *argoKubeWorkflowClusterTemplateServiceClient) UpdateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateUpdateRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	return a.delegate.UpdateClusterWorkflowTemplate(ctx, req)
}

func (a *argoKubeWorkflowClusterTemplateServiceClient) DeleteClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateDeleteRequest, opts ...grpc.CallOption) (*clusterworkflowtmplpkg.ClusterWorkflowTemplateDeleteResponse, error) {
	return a.delegate.DeleteClusterWorkflowTemplate(ctx, req)
}

func (a *argoKubeWorkflowClusterTemplateServiceClient) LintClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateLintRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	return a.delegate.LintClusterWorkflowTemplate(ctx, req)
}
