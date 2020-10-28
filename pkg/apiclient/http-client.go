package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type httpClient struct {
	baseUrl      string
	authSupplier func() string
}

// ArchivedWorkflowService

func (h *httpClient) ListArchivedWorkflows(_ context.Context, in *workflowarchivepkg.ListArchivedWorkflowsRequest, _ ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	out := &wfv1.WorkflowList{}
	return out, h.Get(out, "/api/v1/archived-workflows")
}

func (h *httpClient) GetArchivedWorkflow(_ context.Context, in *workflowarchivepkg.GetArchivedWorkflowRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Get(out, "/api/v1/archived-workflows/{uid}", in.Uid)
}

func (h *httpClient) DeleteArchivedWorkflow(_ context.Context, in *workflowarchivepkg.DeleteArchivedWorkflowRequest, _ ...grpc.CallOption) (*workflowarchivepkg.ArchivedWorkflowDeletedResponse, error) {
	out := &workflowarchivepkg.ArchivedWorkflowDeletedResponse{}
	return out, h.Delete("/api/v1/archived-workflows/{uid}", in.Uid)
}

// ClusterWorkflowTemplateService

func (h *httpClient) CreateClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/cluster-workflow-templates")

}

func (h *httpClient) GetClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Get(out, "/api/v1/cluster-workflow-templates/{name}", in.Name)

}

func (h *httpClient) ListClusterWorkflowTemplates(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateListRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplateList, error) {
	out := &wfv1.ClusterWorkflowTemplateList{}
	return out, h.Get(out, "/api/v1/cluster-workflow-templates")
}

func (h *httpClient) UpdateClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/cluster-workflow-templates/{name}", in.Name)

}

func (h *httpClient) DeleteClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse, error) {
	out := &clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse{}
	return out, h.Delete("/api/v1/cluster-workflow-templates/{name}", in.Name)
}

func (h *httpClient) LintClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/cluster-workflow-templates/lint")
}

// WorkflowTemplateService

func (h *httpClient) CreateWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/workflow-templates/{namespace}", in.Namespace)
}

func (h *httpClient) GetWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Get(out, "/api/v1/workflow-templates/{namespace}/{name}", in.Namespace, in.Name)
}

func (h *httpClient) ListWorkflowTemplates(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplateList, error) {
	out := &wfv1.WorkflowTemplateList{}
	return out, h.Get(out, "/api/v1/workflow-templates/{namespace}", in.Namespace)
}

func (h *httpClient) UpdateWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/workflow-templates/{namespace}/{name}", in.Namespace, in.Name)
}

func (h *httpClient) DeleteWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	out := &workflowtemplatepkg.WorkflowTemplateDeleteResponse{}
	return out, h.Delete("/api/v1/workflow-templates/{namespace}/{name}", in.Namespace, in.Name)
}

func (h *httpClient) LintWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/workflow-templates/{namespace}/lint", in.Namespace)
}

// CronWorkflowService

func (h *httpClient) LintCronWorkflow(_ context.Context, in *cronworkflowpkg.LintCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Post(in, out, "/api/v1/cron-workflows/{namespace}/lint", in.Namespace)
}

func (h *httpClient) CreateCronWorkflow(_ context.Context, in *cronworkflowpkg.CreateCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Post(in, out, "/api/v1/cron-workflows/{namespace}", in.Namespace)
}

func (h *httpClient) ListCronWorkflows(_ context.Context, in *cronworkflowpkg.ListCronWorkflowsRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflowList, error) {
	out := &wfv1.CronWorkflowList{}
	return out, h.Get(out, "/api/v1/cron-workflows/{namespace}", in.Namespace)
}

func (h *httpClient) GetCronWorkflow(_ context.Context, in *cronworkflowpkg.GetCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Get(out, "/api/v1/cron-workflows/{namespace}/{name}", in.Namespace, in.Name)
}

func (h *httpClient) UpdateCronWorkflow(_ context.Context, in *cronworkflowpkg.UpdateCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Put(in, out, "/api/v1/cron-workflows/{namespace}/{name}", in.Namespace, in.Name)
}

func (h *httpClient) DeleteCronWorkflow(_ context.Context, in *cronworkflowpkg.DeleteCronWorkflowRequest, _ ...grpc.CallOption) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	return &cronworkflowpkg.CronWorkflowDeletedResponse{}, h.Delete("/api/v1/cron-workflows/{namespace}/{name}", in.Namespace, in.Name)
}

// WorkflowService

func (h *httpClient) CreateWorkflow(_ context.Context, in *workflowpkg.WorkflowCreateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}", in.Namespace)
}

func (h *httpClient) GetWorkflow(_ context.Context, in *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Get(out, "/api/v1/workflows/{namespace}/{name}", in.Namespace, in.Name)
}

func (h *httpClient) ListWorkflows(_ context.Context, in *workflowpkg.WorkflowListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	out := &wfv1.WorkflowList{}
	return out, h.Get("/api/v1/workflows/{namespace}", in.Namespace)
}

func (h *httpClient) WatchWorkflows(_ context.Context, in *workflowpkg.WatchWorkflowsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	panic("implement me")
}

func (h *httpClient) WatchEvents(_ context.Context, in *workflowpkg.WatchEventsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchEventsClient, error) {
	panic("implement me")
}

func (h *httpClient) DeleteWorkflow(_ context.Context, in *workflowpkg.WorkflowDeleteRequest, _ ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	return &workflowpkg.WorkflowDeleteResponse{}, h.Delete("/api/v1/workflows/{namespace}/{name}", in.Namespace, in.Name)
}

func (h *httpClient) RetryWorkflow(_ context.Context, in *workflowpkg.WorkflowRetryRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/retry", in.Namespace, in.Name)
}

func (h *httpClient) ResubmitWorkflow(_ context.Context, in *workflowpkg.WorkflowResubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/resubmit", in.Namespace, in.Name)
}

func (h *httpClient) ResumeWorkflow(_ context.Context, in *workflowpkg.WorkflowResumeRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	err := h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/resume", in.Namespace, in.Name)
	return out, err
}

func (h *httpClient) SuspendWorkflow(_ context.Context, in *workflowpkg.WorkflowSuspendRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/suspend", in.Namespace, in.Name)
}

func (h *httpClient) TerminateWorkflow(_ context.Context, in *workflowpkg.WorkflowTerminateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/terimate", in.Namespace, in.Name)
}

func (h *httpClient) StopWorkflow(_ context.Context, in *workflowpkg.WorkflowStopRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/stop", in.Namespace, in.Name)
}

func (h *httpClient) SetWorkflow(_ context.Context, in *workflowpkg.WorkflowSetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/set", in.Namespace, in.Name)
}

func (h *httpClient) LintWorkflow(_ context.Context, in *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/lint", in.Namespace)
}

func (h *httpClient) PodLogs(_ context.Context, in *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	panic("implement me")
}

func (h *httpClient) SubmitWorkflow(_ context.Context, in *workflowpkg.WorkflowSubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}/submit", in.Namespace)
}

// InfoService

func (h *httpClient) GetInfo(context.Context, *infopkg.GetInfoRequest, ...grpc.CallOption) (*infopkg.InfoResponse, error) {
	out := &infopkg.InfoResponse{}
	return out, h.Get(out, "/api/v1/info")
}

func (h *httpClient) GetVersion(context.Context, *infopkg.GetVersionRequest, ...grpc.CallOption) (*wfv1.Version, error) {
	out := &wfv1.Version{}
	return out, h.Get(out, "/api/v1/version")
}

func (h *httpClient) GetUserInfo(context.Context, *infopkg.GetUserInfoRequest, ...grpc.CallOption) (*infopkg.GetUserInfoResponse, error) {
	out := &infopkg.GetUserInfoResponse{}
	return out, h.Get(out, "/api/v1/userinfo")
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

func (h *httpClient) Get(out interface{}, path string, args ...interface{}) error {
	return h.do(nil, out, "GET", path, args)
}

func (h *httpClient) Put(in, out interface{}, path string, args ...interface{}) error {
	return h.do(in, out, "PUT", path, args)
}

func (h *httpClient) Post(in, out interface{}, path string, args ...interface{}) error {
	return h.do(in, out, "POST", path, args)
}
func (h *httpClient) Delete(path string, args ...interface{}) error {
	return h.do(nil, nil, "DELETE", path, args)
}

func (h *httpClient) do(in interface{}, out interface{}, method string, path string, args []interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, h.baseUrl+fmt.Sprintf(regexp.MustCompile("{[^}]+}").ReplaceAllString(path, "%s"), args...), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", h.authSupplier())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = errFromResponse(resp.StatusCode)
	if err != nil {
		return err
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func newHTTPClient(baseUrl string, authSupplier func() string) (context.Context, Client, error) {
	return context.Background(), &httpClient{baseUrl: baseUrl, authSupplier: authSupplier}, nil
}

func errFromResponse(statusCode int) error {
	code, ok := map[int]codes.Code{
		http.StatusOK:                  codes.OK,
		http.StatusNotFound:            codes.NotFound,
		http.StatusConflict:            codes.AlreadyExists,
		http.StatusBadRequest:          codes.InvalidArgument,
		http.StatusMethodNotAllowed:    codes.Unimplemented,
		http.StatusServiceUnavailable:  codes.Unavailable,
		http.StatusPreconditionFailed:  codes.FailedPrecondition,
		http.StatusUnauthorized:        codes.Unauthenticated,
		http.StatusForbidden:           codes.PermissionDenied,
		http.StatusRequestTimeout:      codes.DeadlineExceeded,
		http.StatusGatewayTimeout:      codes.DeadlineExceeded,
		http.StatusInternalServerError: codes.Internal,
	}[statusCode]
	if ok {
		if code == codes.OK {
			return nil
		}
		return status.Error(code, "")
	}
	return status.Error(codes.Internal, fmt.Sprintf("unknown error: %v", statusCode))
}
