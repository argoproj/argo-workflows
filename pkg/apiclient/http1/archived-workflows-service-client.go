package http1

import (
	"context"

	"google.golang.org/grpc"

	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type ArchivedWorkflowsServiceClient = Facade

func (h ArchivedWorkflowsServiceClient) ListArchivedWorkflows(_ context.Context, in *workflowarchivepkg.ListArchivedWorkflowsRequest, _ ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	out := &wfv1.WorkflowList{}
	return out, h.Get(in, out, "/api/v1/archived-workflows")
}

func (h ArchivedWorkflowsServiceClient) GetArchivedWorkflow(_ context.Context, in *workflowarchivepkg.GetArchivedWorkflowRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Get(in, out, "/api/v1/archived-workflows/{uid}")
}

func (h ArchivedWorkflowsServiceClient) DeleteArchivedWorkflow(_ context.Context, in *workflowarchivepkg.DeleteArchivedWorkflowRequest, _ ...grpc.CallOption) (*workflowarchivepkg.ArchivedWorkflowDeletedResponse, error) {
	out := &workflowarchivepkg.ArchivedWorkflowDeletedResponse{}
	return out, h.Delete(in, out, "/api/v1/archived-workflows/{uid}")
}

func (h ArchivedWorkflowsServiceClient) DeleteClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse, error) {
	out := &clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse{}
	return out, h.Delete(in, out, "/api/v1/cluster-workflow-templates/{name}")
}

func (h ArchivedWorkflowsServiceClient) LintClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/cluster-workflow-templates/lint")
}
