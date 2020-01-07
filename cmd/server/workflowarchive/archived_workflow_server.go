package workflowarchive

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/server/auth"
	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
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
	offset, err := strconv.Atoi(options.Continue)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "listOptions.continue must be int")
	}
	items := make(wfv1.Workflows, 0)
	authorizer := auth.NewAuthorizer(ctx)
	// keep trying until we have enough items
	unlimited := limit == 0
	for len(items) < limit || unlimited {
		moreItems, err := w.repo.ListWorkflows(req.Namespace, limit, offset)
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
		if len(moreItems) < limit || unlimited {
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
	wf, err := w.repo.GetWorkflow(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}
	if wf == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	allowed, err := auth.CanI(ctx, "get", "workflow", req.Namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return wf, err
}

func (w *archivedWorkflowServer) ResubmitArchivedWorkflow(ctx context.Context, req *ResubmitArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wf, err := w.GetArchivedWorkflow(ctx, &GetArchivedWorkflowRequest{Namespace: req.Namespace, Uid: req.Uid})
	if err != nil {
		return nil, err
	}
	wf, err = util.FormulateResubmitWorkflow(wf, false)
	if err != nil {
		return nil, err
	}
	wfClient := auth.GetWfClient(ctx)
	wf, err = util.SubmitWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, wf.Namespace, wf, &util.SubmitOpts{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (w *archivedWorkflowServer) DeleteArchivedWorkflow(ctx context.Context, req *DeleteArchivedWorkflowRequest) (*ArchivedWorkflowDeletedResponse, error) {
	wf, err := w.GetArchivedWorkflow(ctx, &GetArchivedWorkflowRequest{Namespace: req.Namespace, Uid: req.Uid})
	if err != nil {
		return nil, err
	}
	allowed, err := auth.CanI(ctx, "delete", "workflow", req.Namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	err = w.repo.DeleteWorkflow(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}
	return &ArchivedWorkflowDeletedResponse{}, nil
}
