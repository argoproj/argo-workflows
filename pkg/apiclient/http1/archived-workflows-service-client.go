package http1

import (
	"context"

	"google.golang.org/grpc"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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

func (h ArchivedWorkflowsServiceClient) ListArchivedWorkflowLabelKeys(_ context.Context, in *workflowarchivepkg.ListArchivedWorkflowLabelKeysRequest, _ ...grpc.CallOption) (*wfv1.LabelKeys, error) {
	out := &wfv1.LabelKeys{}
	return out, h.Get(in, out, "/api/v1/archived-workflows-label-keys")
}

func (h ArchivedWorkflowsServiceClient) ListArchivedWorkflowLabelValues(_ context.Context, in *workflowarchivepkg.ListArchivedWorkflowLabelValuesRequest, _ ...grpc.CallOption) (*wfv1.LabelValues, error) {
	out := &wfv1.LabelValues{}
	return out, h.Get(in, out, "/api/v1/archived-workflows-label-values")
}

func (h ArchivedWorkflowsServiceClient) RetryArchivedWorkflow(_ context.Context, in *workflowarchivepkg.RetryArchivedWorkflowRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/archived-workflows/{uid}/retry")
}

func (h ArchivedWorkflowsServiceClient) ResubmitArchivedWorkflow(_ context.Context, in *workflowarchivepkg.ResubmitArchivedWorkflowRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/archived-workflows/{uid}/resubmit")
}
