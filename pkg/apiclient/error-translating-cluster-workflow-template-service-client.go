package apiclient

import (
	"context"

	"google.golang.org/grpc"

	clusterworkflowtmplpkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	grpcutil "github.com/argoproj/argo/util/grpc"
)

type errorTranslatingWorkflowClusterTemplateServiceClient struct {
	delegate clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient
}

var _ clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient = &errorTranslatingWorkflowClusterTemplateServiceClient{}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) CreateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateCreateRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	template, err := a.delegate.CreateClusterWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) GetClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	template, err := a.delegate.GetClusterWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) ListClusterWorkflowTemplates(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateListRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplateList, error) {
	templates, err := a.delegate.ListClusterWorkflowTemplates(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return templates, err
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) UpdateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateUpdateRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	template, err := a.delegate.UpdateClusterWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) DeleteClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateDeleteRequest, opts ...grpc.CallOption) (*clusterworkflowtmplpkg.ClusterWorkflowTemplateDeleteResponse, error) {
	template, err := a.delegate.DeleteClusterWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}

func (a errorTranslatingWorkflowClusterTemplateServiceClient) LintClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateLintRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	template, err := a.delegate.LintClusterWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}
