package apiclient

import (
	"context"

	"google.golang.org/grpc"

	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v4/workflow/validate"
)

type OfflineClusterWorkflowTemplateServiceClient struct {
	clusterWorkflowTemplateGetter       templateresolution.ClusterWorkflowTemplateGetter
	namespacedWorkflowTemplateGetterMap offlineWorkflowTemplateGetterMap
}

var _ clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient = &OfflineClusterWorkflowTemplateServiceClient{}

func (o OfflineClusterWorkflowTemplateServiceClient) CreateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateCreateRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	return nil, ErrOffline
}

func (o OfflineClusterWorkflowTemplateServiceClient) GetClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	return nil, ErrOffline
}

func (o OfflineClusterWorkflowTemplateServiceClient) ListClusterWorkflowTemplates(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateListRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplateList, error) {
	return nil, ErrOffline
}

func (o OfflineClusterWorkflowTemplateServiceClient) UpdateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateUpdateRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	return nil, ErrOffline
}

func (o OfflineClusterWorkflowTemplateServiceClient) DeleteClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateDeleteRequest, opts ...grpc.CallOption) (*clusterworkflowtmplpkg.ClusterWorkflowTemplateDeleteResponse, error) {
	return nil, ErrOffline
}

func (o OfflineClusterWorkflowTemplateServiceClient) LintClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateLintRequest, opts ...grpc.CallOption) (*v1alpha1.ClusterWorkflowTemplate, error) {
	err := validate.ValidateClusterWorkflowTemplate(ctx, nil, o.clusterWorkflowTemplateGetter, req.Template, nil, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, err
	}
	return req.Template, nil
}
