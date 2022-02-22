package http1

import (
	"context"

	"google.golang.org/grpc"

	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type CronWorkflowServiceClient = Facade

func (h CronWorkflowServiceClient) LintCronWorkflow(_ context.Context, in *cronworkflowpkg.LintCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Post(in, out, "/api/v1/cron-workflows/{namespace}/lint")
}

func (h CronWorkflowServiceClient) CreateCronWorkflow(_ context.Context, in *cronworkflowpkg.CreateCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Post(in, out, "/api/v1/cron-workflows/{namespace}")
}

func (h CronWorkflowServiceClient) ListCronWorkflows(_ context.Context, in *cronworkflowpkg.ListCronWorkflowsRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflowList, error) {
	out := &wfv1.CronWorkflowList{}
	return out, h.Get(in, out, "/api/v1/cron-workflows/{namespace}")
}

func (h CronWorkflowServiceClient) GetCronWorkflow(_ context.Context, in *cronworkflowpkg.GetCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Get(in, out, "/api/v1/cron-workflows/{namespace}/{name}")
}

func (h CronWorkflowServiceClient) UpdateCronWorkflow(_ context.Context, in *cronworkflowpkg.UpdateCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Put(in, out, "/api/v1/cron-workflows/{namespace}/{name}")
}

func (h Facade) ResumeCronWorkflow(ctx context.Context, in *cronworkflowpkg.CronWorkflowResumeRequest, opts ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Put(in, out, "/api/v1/cron-workflows/{namespace}/{name}/resume")
}

func (h Facade) SuspendCronWorkflow(ctx context.Context, in *cronworkflowpkg.CronWorkflowSuspendRequest, opts ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	out := &wfv1.CronWorkflow{}
	return out, h.Put(in, out, "/api/v1/cron-workflows/{namespace}/{name}/suspend")
}

func (h CronWorkflowServiceClient) DeleteCronWorkflow(_ context.Context, in *cronworkflowpkg.DeleteCronWorkflowRequest, _ ...grpc.CallOption) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	out := &cronworkflowpkg.CronWorkflowDeletedResponse{}
	return out, h.Delete(in, out, "/api/v1/cron-workflows/{namespace}/{name}")
}
