package http1

import (
	"context"

	"google.golang.org/grpc"

	workflowarchivelabelpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchivelabel"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type ArchivedWorkflowLabelServiceClient = Facade

func (h ArchivedWorkflowLabelServiceClient) ListArchivedWorkflowLabel(_ context.Context, in *workflowarchivelabelpkg.ListArchivedWorkflowLabelRequest, _ ...grpc.CallOption) (*wfv1.LabelKeys, error) {
	out := &wfv1.LabelKeys{}
	return out, h.Get(in, out, "/api/v1/archived-workflows-labels")
}

func (h ArchivedWorkflowLabelServiceClient) GetArchivedWorkflowLabel(_ context.Context, in *workflowarchivelabelpkg.GetArchivedWorkflowLabelRequest, _ ...grpc.CallOption) (*wfv1.Labels, error) {
	out := &wfv1.Labels{}
	return out, h.Get(in, out, "/api/v1/archived-workflows-labels/{key}")
}
