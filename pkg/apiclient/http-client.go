package apiclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"

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
	return out, h.Get(out, "/api/v1/archived-workflows?%s", queryParams(in))
}

func queryParams(in interface{}) string {
	out := toMap(in)
	var params []string
	for k, v := range out {
		for s, i := range toMap(v) {
			params = append(params, fmt.Sprintf("%s.%s=%v", k, s, i))
		}
	}
	return strings.Join(params, "&")
}

func toMap(in interface{}) map[string]interface{} {
	data, _ := json.Marshal(in)
	out := make(map[string]interface{})
	_ = json.Unmarshal(data, &out)
	return out
}

func (h *httpClient) GetArchivedWorkflow(_ context.Context, in *workflowarchivepkg.GetArchivedWorkflowRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Get(out, "/api/v1/archived-workflows/{uid}", in.Uid)
}

func (h *httpClient) DeleteArchivedWorkflow(_ context.Context, in *workflowarchivepkg.DeleteArchivedWorkflowRequest, _ ...grpc.CallOption) (*workflowarchivepkg.ArchivedWorkflowDeletedResponse, error) {
	out := &workflowarchivepkg.ArchivedWorkflowDeletedResponse{}
	return out, h.Delete("/api/v1/archived-workflows/{uid}?%s", in.Uid, queryParams(in))
}

// ClusterWorkflowTemplateService

func (h *httpClient) CreateClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/cluster-workflow-templates?%s", queryParams(in))

}

func (h *httpClient) GetClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Get(out, "/api/v1/cluster-workflow-templates/{name}?s", in.Name, queryParams(in))

}

func (h *httpClient) ListClusterWorkflowTemplates(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateListRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplateList, error) {
	out := &wfv1.ClusterWorkflowTemplateList{}
	return out, h.Get(out, "/api/v1/cluster-workflow-templates?%s", queryParams(in))
}

func (h *httpClient) UpdateClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/cluster-workflow-templates/{name}?%s", in.Name, queryParams(in))

}

func (h *httpClient) DeleteClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse, error) {
	out := &clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse{}
	return out, h.Delete("/api/v1/cluster-workflow-templates/{name}?%s", in.Name, queryParams(in))
}

func (h *httpClient) LintClusterWorkflowTemplate(_ context.Context, in *clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	out := &wfv1.ClusterWorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/cluster-workflow-templates/lint?%s", queryParams(in))
}

// WorkflowTemplateService

func (h *httpClient) CreateWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/workflow-templates/{namespace}?%s", in.Namespace, queryParams(in))
}

func (h *httpClient) GetWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Get(out, "/api/v1/workflow-templates/{namespace}/{name}?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) ListWorkflowTemplates(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplateList, error) {
	out := &wfv1.WorkflowTemplateList{}
	return out, h.Get(out, "/api/v1/workflow-templates/{namespace}?%s", in.Namespace, queryParams(in))
}

func (h *httpClient) UpdateWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/workflow-templates/{namespace}/{name}?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) DeleteWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	out := &workflowtemplatepkg.WorkflowTemplateDeleteResponse{}
	return out, h.Delete("/api/v1/workflow-templates/{namespace}/{name}?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) LintWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/workflow-templates/{namespace}/lint", in.Namespace)
}

// CronWorkflowService

func (h *httpClient) LintCronWorkflow(_ context.Context, in *cronworkflowpkg.LintCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Post(in, out, "/api/v1/cron-workflows/{namespace}/lint?%s", in.Namespace, queryParams(in))
}

func (h *httpClient) CreateCronWorkflow(_ context.Context, in *cronworkflowpkg.CreateCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Post(in, out, "/api/v1/cron-workflows/{namespace}?%s", in.Namespace, queryParams(in))
}

func (h *httpClient) ListCronWorkflows(_ context.Context, in *cronworkflowpkg.ListCronWorkflowsRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflowList, error) {
	out := &wfv1.CronWorkflowList{}
	return out, h.Get(out, "/api/v1/cron-workflows/{namespace}?%s", in.Namespace, queryParams(in))
}

func (h *httpClient) GetCronWorkflow(_ context.Context, in *cronworkflowpkg.GetCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Get(out, "/api/v1/cron-workflows/{namespace}/{name}?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) UpdateCronWorkflow(_ context.Context, in *cronworkflowpkg.UpdateCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Put(in, out, "/api/v1/cron-workflows/{namespace}/{name}?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) DeleteCronWorkflow(_ context.Context, in *cronworkflowpkg.DeleteCronWorkflowRequest, _ ...grpc.CallOption) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	return &cronworkflowpkg.CronWorkflowDeletedResponse{}, h.Delete("/api/v1/cron-workflows/{namespace}/{name}?%s", in.Namespace, in.Name, queryParams(in))
}

// WorkflowService

func (h *httpClient) CreateWorkflow(_ context.Context, in *workflowpkg.WorkflowCreateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}?%s", in.Namespace, queryParams(in))
}

func (h *httpClient) GetWorkflow(_ context.Context, in *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Get(out, "/api/v1/workflows/{namespace}/{name}?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) ListWorkflows(_ context.Context, in *workflowpkg.WorkflowListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	out := &wfv1.WorkflowList{}
	return out, h.Get("/api/v1/workflows/{namespace}?%s", in.Namespace, queryParams(in))
}

type httpWatchClient struct {
	abstractIntermediary
	reader *bufio.Reader
}

func (f *httpWatchClient) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	data, err := f.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	out := &workflowpkg.WorkflowWatchEvent{}
	return out, json.Unmarshal(data, out)
}

func (h *httpClient) WatchWorkflows(ctx context.Context, in *workflowpkg.WatchWorkflowsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	reader, err := h.eventStreamReader("/api/v1/workflow-events/{namespace}?%s", in.Namespace, queryParams(in))
	if err != nil {
		return nil, err
	}
	return &httpWatchClient{abstractIntermediary: newAbstractIntermediary(ctx), reader: reader}, nil
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
	reader, err := h.eventStreamReader("/api/v1/stream/events/{namespace}?%s", in.Namespace, queryParams(in.ListOptions))
	if err != nil {
		return nil, err
	}
	return &httpEventWatchClient{abstractIntermediary: newAbstractIntermediary(ctx), reader: reader}, nil
}

func (h *httpClient) DeleteWorkflow(_ context.Context, in *workflowpkg.WorkflowDeleteRequest, _ ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	return &workflowpkg.WorkflowDeleteResponse{}, h.Delete("/api/v1/workflows/{namespace}/{name}?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) RetryWorkflow(_ context.Context, in *workflowpkg.WorkflowRetryRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/retry?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) ResubmitWorkflow(_ context.Context, in *workflowpkg.WorkflowResubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/resubmit?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) ResumeWorkflow(_ context.Context, in *workflowpkg.WorkflowResumeRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	err := h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/resume?%s", in.Namespace, in.Name, queryParams(in))
	return out, err
}

func (h *httpClient) SuspendWorkflow(_ context.Context, in *workflowpkg.WorkflowSuspendRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/suspend?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) TerminateWorkflow(_ context.Context, in *workflowpkg.WorkflowTerminateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/terminate?%v", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) StopWorkflow(_ context.Context, in *workflowpkg.WorkflowStopRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/stop?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) SetWorkflow(_ context.Context, in *workflowpkg.WorkflowSetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/set?%s", in.Namespace, in.Name, queryParams(in))
}

func (h *httpClient) LintWorkflow(_ context.Context, in *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/lint?%s", in.Namespace, queryParams(in))
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
	reader, err := h.eventStreamReader("/api/v1/workflows/{namespace}/{name}/{podName}/log?%s", in.Namespace, in.Name, in.PodName, queryParams(in))
	if err != nil {
		return nil, err
	}
	return &httpPodLogsClient{abstractIntermediary: newAbstractIntermediary(ctx), reader: reader}, nil
}

func (h *httpClient) SubmitWorkflow(_ context.Context, in *workflowpkg.WorkflowSubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}/submit?%s", in.Namespace, queryParams(in))
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

func (h *httpClient) eventStreamReader(path string, args ...interface{}) (*bufio.Reader, error) {
	req, err := http.NewRequest("GET", h.apiURL(path, args...), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", h.authSupplier())
	req.Close = true
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	err = errFromResponse(resp.StatusCode)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(resp.Body)
	return reader, nil
}

func (h *httpClient) do(in interface{}, out interface{}, method string, path string, args []interface{}) error {
	var data []byte
	var err error
	if in != nil {
		data, err = json.Marshal(in)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, h.apiURL(path, args...), bytes.NewReader(data))
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
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	} else {
		return nil
	}
}

func (h *httpClient) apiURL(path string, args ...interface{}) string {
	return h.baseUrl + fmt.Sprintf(regexp.MustCompile("{[^}]+}").ReplaceAllString(path, "%s"), args...)
}

func newHTTPClient(baseUrl string, authSupplier func() string) (context.Context, Client, error) {
	return context.Background(), &httpClient{baseUrl: baseUrl, authSupplier: authSupplier}, nil
}

func errFromResponse(statusCode int) error {
	if statusCode == http.StatusOK {
		return nil
	}
	code, ok := map[int]codes.Code{
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
		return status.Error(code, "")
	}
	return status.Error(codes.Internal, fmt.Sprintf("unknown error: %v", statusCode))
}
