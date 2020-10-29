package apiclient

import (
	"bufio"
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apiclient/http"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type httpClient struct {
	http.Facade
}

// ArchivedWorkflowService

func (h *httpClient) ListArchivedWorkflows(_ context.Context, in *workflowarchivepkg.ListArchivedWorkflowsRequest, _ ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	out := &wfv1.WorkflowList{}
	return out, h.Get(in, out, "/api/v1/archived-workflows")
}

func (h *httpClient) GetArchivedWorkflow(_ context.Context, in *workflowarchivepkg.GetArchivedWorkflowRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Get(in, out, "/api/v1/archived-workflows/{uid}")
}

func (h *httpClient) DeleteArchivedWorkflow(_ context.Context, in *workflowarchivepkg.DeleteArchivedWorkflowRequest, _ ...grpc.CallOption) (*workflowarchivepkg.ArchivedWorkflowDeletedResponse, error) {
	out := &workflowarchivepkg.ArchivedWorkflowDeletedResponse{}
	return out, h.Delete(in, out, "/api/v1/archived-workflows/{uid}")
}

// ClusterWorkflowTemplateService

func (h *httpClient) CreateClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/cluster-workflow-templates")

}

func (h *httpClient) GetClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Get(in, out, "/api/v1/cluster-workflow-templates/{name}")

}

func (h *httpClient) ListClusterWorkflowTemplates(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateListRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplateList, error) {
	out := &wfv1.ClusterWorkflowTemplateList{}
	return out, h.Get(in, out, "/api/v1/cluster-workflow-templates")
}

func (h *httpClient) UpdateClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/cluster-workflow-templates/{name}")

}

func (h *httpClient) DeleteClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse, error) {
	out := &clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse{}
	return out, h.Delete(in, out, "/api/v1/cluster-workflow-templates/{name}")
}

func (h *httpClient) LintClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/cluster-workflow-templates/lint")
}

// WorkflowTemplateService

func (h *httpClient) CreateWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/workflow-templates/{namespace}")
}

func (h *httpClient) GetWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Get(in, out, "/api/v1/workflow-templates/{namespace}/{name}")
}

func (h *httpClient) ListWorkflowTemplates(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplateList, error) {
	out := &wfv1.WorkflowTemplateList{}
	return out, h.Get(in, out, "/api/v1/workflow-templates/{namespace}")
}

func (h *httpClient) UpdateWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/workflow-templates/{namespace}/{name}")
}

func (h *httpClient) DeleteWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	out := &workflowtemplatepkg.WorkflowTemplateDeleteResponse{}
	return out, h.Delete(in, out, "/api/v1/workflow-templates/{namespace}/{name}")
}

func (h *httpClient) LintWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/workflow-templates/{namespace}/lint")
}

// CronWorkflowService

func (h *httpClient) LintCronWorkflow(_ context.Context, in *cronworkflowpkg.LintCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Post(in, out, "/api/v1/cron-workflows/{namespace}/lint")
}

func (h *httpClient) CreateCronWorkflow(_ context.Context, in *cronworkflowpkg.CreateCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Post(in, out, "/api/v1/cron-workflows/{namespace}")
}

func (h *httpClient) ListCronWorkflows(_ context.Context, in *cronworkflowpkg.ListCronWorkflowsRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflowList, error) {
	out := &wfv1.CronWorkflowList{}
	return out, h.Get(in, out, "/api/v1/cron-workflows/{namespace}")
}

func (h *httpClient) GetCronWorkflow(_ context.Context, in *cronworkflowpkg.GetCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Get(in, out, "/api/v1/cron-workflows/{namespace}/{name}")
}

func (h *httpClient) UpdateCronWorkflow(_ context.Context, in *cronworkflowpkg.UpdateCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Put(in, out, "/api/v1/cron-workflows/{namespace}/{name}")
}

func (h *httpClient) DeleteCronWorkflow(_ context.Context, in *cronworkflowpkg.DeleteCronWorkflowRequest, _ ...grpc.CallOption) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	out := &cronworkflowpkg.CronWorkflowDeletedResponse{}
	return out, h.Delete(in, out, "/api/v1/cron-workflows/{namespace}/{name}")
}

// WorkflowService

func (h *httpClient) CreateWorkflow(_ context.Context, in *workflowpkg.WorkflowCreateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}")
}

func (h *httpClient) GetWorkflow(_ context.Context, in *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Get(in, out, "/api/v1/workflows/{namespace}/{name}")
}

func (h *httpClient) ListWorkflows(_ context.Context, in *workflowpkg.WorkflowListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	out := &wfv1.WorkflowList{}
	return out, h.Get(in, out, "/api/v1/workflows/{namespace}")
}

type httpWatchWorkflowsClient struct {
	abstractIntermediary
	reader *bufio.Reader
}

const prefixLength = len("data: ")
func (f *httpWatchWorkflowsClient) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	for {
		data, err := f.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		if len(data) <= prefixLength {
			continue
		}
		out := &workflowpkg.WorkflowWatchEvent{}
		return out, json.Unmarshal(data[prefixLength:], out)
	}
}

func (h *httpClient) WatchWorkflows(ctx context.Context, in *workflowpkg.WatchWorkflowsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	reader, err := h.EventStreamReader(in, "/api/v1/workflow-events/{namespace}")
	if err != nil {
		return nil, err
	}
	return &httpWatchWorkflowsClient{abstractIntermediary: newAbstractIntermediary(ctx), reader: reader}, nil
}

type httpEventWatchClient struct {
	abstractIntermediary
	reader *bufio.Reader
}

func (f *httpEventWatchClient) Recv() (*corev1.Event, error) {
	data, err := f.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	out := &corev1.Event{}
	return out, json.Unmarshal(data, out)
}

func (h *httpClient) WatchEvents(ctx context.Context, in *workflowpkg.WatchEventsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchEventsClient, error) {
	reader, err := h.EventStreamReader(in, "/api/v1/stream/events/{namespace}")
	if err != nil {
		return nil, err
	}
	return &httpEventWatchClient{abstractIntermediary: newAbstractIntermediary(ctx), reader: reader}, nil
}

func (h *httpClient) DeleteWorkflow(_ context.Context, in *workflowpkg.WorkflowDeleteRequest, _ ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	out := &workflowpkg.WorkflowDeleteResponse{}
	return out, h.Delete(in, out, "/api/v1/workflows/{namespace}/{name}")
}

func (h *httpClient) RetryWorkflow(_ context.Context, in *workflowpkg.WorkflowRetryRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/retry")
}

func (h *httpClient) ResubmitWorkflow(_ context.Context, in *workflowpkg.WorkflowResubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/resubmit")
}

func (h *httpClient) ResumeWorkflow(_ context.Context, in *workflowpkg.WorkflowResumeRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	err := h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/resume")
	return out, err
}

func (h *httpClient) SuspendWorkflow(_ context.Context, in *workflowpkg.WorkflowSuspendRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/suspend")
}

func (h *httpClient) TerminateWorkflow(_ context.Context, in *workflowpkg.WorkflowTerminateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/terminate")
}

func (h *httpClient) StopWorkflow(_ context.Context, in *workflowpkg.WorkflowStopRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/stop")
}

func (h *httpClient) SetWorkflow(_ context.Context, in *workflowpkg.WorkflowSetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/set")
}

func (h *httpClient) LintWorkflow(_ context.Context, in *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}/lint")
}

type httpPodLogsClient struct {
	abstractIntermediary
	reader *bufio.Reader
}

func (f *httpPodLogsClient) Recv() (*workflowpkg.LogEntry, error) {
	data, err := f.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	out := &workflowpkg.LogEntry{}
	return out, json.Unmarshal(data, out)
}

func (h *httpClient) PodLogs(ctx context.Context, in *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	reader, err := h.EventStreamReader(in, "/api/v1/workflows/{namespace}/{name}/{podName}/log")
	if err != nil {
		return nil, err
	}
	return &httpPodLogsClient{abstractIntermediary: newAbstractIntermediary(ctx), reader: reader}, nil
}

func (h *httpClient) SubmitWorkflow(_ context.Context, in *workflowpkg.WorkflowSubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}/submit")
}

// InfoService

func (h *httpClient) GetInfo(_ context.Context, in *infopkg.GetInfoRequest, _ ...grpc.CallOption) (*infopkg.InfoResponse, error) {
	out := &infopkg.InfoResponse{}
	return out, h.Get(in, out, "/api/v1/info")
}

func (h *httpClient) GetVersion(_ context.Context, in *infopkg.GetVersionRequest, _ ...grpc.CallOption) (*wfv1.Version, error) {
	out := &wfv1.Version{}
	return out, h.Get(in, out, "/api/v1/version")
}

func (h *httpClient) GetUserInfo(_ context.Context, in *infopkg.GetUserInfoRequest, _ ...grpc.CallOption) (*infopkg.GetUserInfoResponse, error) {
	out := &infopkg.GetUserInfoResponse{}
	return out, h.Get(in, out, "/api/v1/userinfo")
}

// all
func (h *httpClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return h, nil
}

func (h *httpClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return h
}

func (h *httpClient) NewCronWorkflowServiceClient() cronworkflowpkg.CronWorkflowServiceClient {
	return h
}

func (h *httpClient) NewWorkflowTemplateServiceClient() workflowtemplatepkg.WorkflowTemplateServiceClient {
	return h
}

func (h *httpClient) NewClusterWorkflowTemplateServiceClient() clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient {
	return h
}

func (h *httpClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return h, nil
}

func newHTTPClient(baseUrl string, authSupplier func() string) (context.Context, Client, error) {
	return context.Background(), &httpClient{Facade: http.NewFacade(baseUrl, authSupplier())}, nil
}
