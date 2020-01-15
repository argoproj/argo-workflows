package workflowarchive

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/server/auth"
	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type archivedWorkflowServer struct {
	repo sqldb.WorkflowArchive
}

func NewWorkflowArchiveServer(repo sqldb.WorkflowArchive) ArchivedWorkflowServiceServer {
	return &archivedWorkflowServer{repo: repo}
}

func (w *archivedWorkflowServer) ListArchivedWorkflows(ctx context.Context, req *ListArchivedWorkflowsRequest) (*wfv1.WorkflowList, error) {
	options := req.ListOptions
	if options == nil {
		options = &metav1.ListOptions{}
	}
	if options.Continue == "" {
		options.Continue = "0"
	}
	limit := int(options.Limit)
	if limit == 0 {
		limit = 10
	}
	offset, err := strconv.Atoi(options.Continue)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "listOptions.continue must be int")
	}
	if offset < 0 {
		return nil, status.Error(codes.InvalidArgument, "listOptions.continue must >= 0")
	}
	namespace := ""
	if strings.HasPrefix(options.FieldSelector, "metadata.namespace=") {
		namespace = strings.TrimPrefix(options.FieldSelector, "metadata.namespace=")
	}

	items := make(wfv1.Workflows, 0)
	authorizer := auth.NewAuthorizer(ctx)
	// keep trying until we have enough
	for len(items) < limit {
		moreItems, err := w.repo.ListWorkflows(namespace, limit, offset)
		if err != nil {
			return nil, err
		}
		for _, wf := range moreItems {
			allowed, err := authorizer.CanI("get", "workflow", wf.Namespace, wf.Name)
			if err != nil {
				return nil, err
			}
			if allowed {
				items = append(items, wf)
			}
		}
		if len(moreItems) < limit {
			break
		}
		offset = offset + limit
	}
	meta := metav1.ListMeta{}
	if len(items) >= limit {
		meta.Continue = fmt.Sprintf("%v", offset)
	}
	sort.Sort(items)
	return &wfv1.WorkflowList{ListMeta: meta, Items: items}, nil
}

func (w *archivedWorkflowServer) GetArchivedWorkflow(ctx context.Context, req *GetArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wf, err := w.repo.GetWorkflow(req.Uid)
	if err != nil {
		return nil, err
	}
	if wf == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	allowed, err := auth.CanI(ctx, "get", "workflow", wf.Namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return wf, err
}

func (w *archivedWorkflowServer) DeleteArchivedWorkflow(ctx context.Context, req *DeleteArchivedWorkflowRequest) (*ArchivedWorkflowDeletedResponse, error) {
	wf, err := w.GetArchivedWorkflow(ctx, &GetArchivedWorkflowRequest{Uid: req.Uid})
	if err != nil {
		return nil, err
	}
	allowed, err := auth.CanI(ctx, "delete", "workflow", wf.Namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	err = w.repo.DeleteWorkflow(req.Uid)
	if err != nil {
		return nil, err
	}
	return &ArchivedWorkflowDeletedResponse{}, nil
}
