package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

// WorkflowService

func (h *httpClient) CreateWorkflow(ctx context.Context, in *workflowpkg.WorkflowCreateRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (h *httpClient) GetWorkflow(_ context.Context, in *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	v := &wfv1.Workflow{}
	return v, h.Get(v, "/api/v1/workflows/{namespace}/{name}", in.Namespace, in.Name)
}

func (h *httpClient) ListWorkflows(ctx context.Context, in *workflowpkg.WorkflowListRequest, opts ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	panic("implement me")
}

func (h *httpClient) WatchWorkflows(ctx context.Context, in *workflowpkg.WatchWorkflowsRequest, opts ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	panic("implement me")
}

func (h *httpClient) WatchEvents(ctx context.Context, in *workflowpkg.WatchEventsRequest, opts ...grpc.CallOption) (workflowpkg.WorkflowService_WatchEventsClient, error) {
	panic("implement me")
}

func (h *httpClient) DeleteWorkflow(ctx context.Context, in *workflowpkg.WorkflowDeleteRequest, opts ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	panic("implement me")
}

func (h *httpClient) RetryWorkflow(ctx context.Context, in *workflowpkg.WorkflowRetryRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (h *httpClient) ResubmitWorkflow(ctx context.Context, in *workflowpkg.WorkflowResubmitRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (h *httpClient) ResumeWorkflow(ctx context.Context, in *workflowpkg.WorkflowResumeRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (h *httpClient) SuspendWorkflow(ctx context.Context, in *workflowpkg.WorkflowSuspendRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (h *httpClient) TerminateWorkflow(ctx context.Context, in *workflowpkg.WorkflowTerminateRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (h *httpClient) StopWorkflow(ctx context.Context, in *workflowpkg.WorkflowStopRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (h *httpClient) SetWorkflow(ctx context.Context, in *workflowpkg.WorkflowSetRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (h *httpClient) LintWorkflow(ctx context.Context, in *workflowpkg.WorkflowLintRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (h *httpClient) PodLogs(ctx context.Context, in *workflowpkg.WorkflowLogRequest, opts ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	panic("implement me")
}

func (h *httpClient) SubmitWorkflow(ctx context.Context, in *workflowpkg.WorkflowSubmitRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	panic("implement me")
}

// InfoService
func (h *httpClient) GetInfo(context.Context, *infopkg.GetInfoRequest, ...grpc.CallOption) (*infopkg.InfoResponse, error) {
	v := &infopkg.InfoResponse{}
	return v, h.Get("/api/v1/info", v)
}

func (h *httpClient) GetVersion(context.Context, *infopkg.GetVersionRequest, ...grpc.CallOption) (*wfv1.Version, error) {
	v := &wfv1.Version{}
	return v, h.Get("/api/v1/version", v)
}

func (h *httpClient) GetUserInfo(context.Context, *infopkg.GetUserInfoRequest, ...grpc.CallOption) (*infopkg.GetUserInfoResponse, error) {
	v := &infopkg.GetUserInfoResponse{}
	return v, h.Get("/api/v1/userinfo", v)
}

// all
func (h *httpClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	panic("implement me")
}

func (h *httpClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return h
}

func (h *httpClient) NewCronWorkflowServiceClient() cronworkflowpkg.CronWorkflowServiceClient {
	panic("implement me")
}

func (h *httpClient) NewWorkflowTemplateServiceClient() workflowtemplatepkg.WorkflowTemplateServiceClient {
	panic("implement me")
}

func (h *httpClient) NewClusterWorkflowTemplateServiceClient() clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient {
	panic("implement me")
}

func (h *httpClient) Get(v interface{}, path string, args ...interface{} ) error {
	req, err := http.NewRequest("GET", h.baseUrl+fmt.Sprintf(path, args...), nil)
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
	return json.NewDecoder(resp.Body).Decode(v)
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

func (h *httpClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return h, nil
}

func newHTTPClient(baseUrl string, authSupplier func() string) (context.Context, Client, error) {
	return context.Background(), &httpClient{baseUrl: baseUrl, authSupplier: authSupplier}, nil
}
