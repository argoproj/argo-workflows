package apiclient

import (
	"context"

	"google.golang.org/grpc"

	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	grpcutil "github.com/argoproj/argo-workflows/v4/util/grpc"
)

type errorTranslatingWorkflowClusterTemplateServiceClient struct {
	delegate clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient
}

var _ clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient = &errorTranslatingWorkflowClusterTemplateServiceClient{}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) CreateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateCreateRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	template, err := a.delegate.CreateClusterWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) GetClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	template, err := a.delegate.GetClusterWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) ListClusterWorkflowTemplates(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateListRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplateList, error) {
	templates, err := a.delegate.ListClusterWorkflowTemplates(ctx, req)
	return templates, grpcutil.TranslateError(err)
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) UpdateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateUpdateRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	template, err := a.delegate.UpdateClusterWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) DeleteClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateDeleteRequest, opts ...grpc.CallOption) (*clusterworkflowtmplpkg.ClusterWorkflowTemplateDeleteResponse, error) {
	template, err := a.delegate.DeleteClusterWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) LintClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateLintRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	template, err := a.delegate.LintClusterWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}
