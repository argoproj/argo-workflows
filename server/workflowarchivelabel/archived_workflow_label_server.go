package workflowarchivelabel

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	workflowarchivelabelpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchivelabel"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type archivedWorkflowLabelServer struct {
	wfArchive sqldb.WorkflowArchive
}

// NewWorkflowArchiveLabelServer returns a new archivedWorkflowLabelServer
func NewWorkflowArchiveLabelServer(wfArchive sqldb.WorkflowArchive) workflowarchivelabelpkg.ArchivedWorkflowLabelServiceServer {
	return &archivedWorkflowLabelServer{wfArchive: wfArchive}
}

func (w *archivedWorkflowLabelServer) ListArchivedWorkflowLabel(ctx context.Context, req *workflowarchivelabelpkg.ListArchivedWorkflowLabelRequest) (*wfv1.LabelKeys, error) {
	labelkeys, err := w.wfArchive.ListWorkflowsLabelKey()
	if err != nil {
		return nil, err
	}
	return labelkeys, nil
}

func (w *archivedWorkflowLabelServer) GetArchivedWorkflowLabel(ctx context.Context, req *workflowarchivelabelpkg.GetArchivedWorkflowLabelRequest) (*wfv1.Labels, error) {
	options := req.ListOptions

	key := ""
	for _, selector := range strings.Split(options.FieldSelector, ",") {
		if len(selector) == 0 {
			continue
		}
		if strings.HasPrefix(selector, "key=") {
			key = strings.TrimPrefix(selector, "key=")
		} else {
			return nil, fmt.Errorf("unsupported requirement %s", selector)
		}
	}

	labels, err := w.wfArchive.GetWorkflowLabel(key)
	if err != nil {
		return nil, err
	}
	if labels == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	return labels, err
}
