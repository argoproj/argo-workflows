package workflowhistory

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/server/auth"
	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
)

type workflowHistoryServer struct {
	repo sqldb.WorkflowHistoryRepository
}

func NewWorkflowHistoryServer(repo sqldb.WorkflowHistoryRepository) WorkflowHistoryServiceServer {
	return &workflowHistoryServer{repo: repo}
}

func (w *workflowHistoryServer) ListWorkflowHistory(ctx context.Context, req *WorkflowHistoryListRequest) (*wfv1.WorkflowList, error) {
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
	allItems, err := w.repo.ListWorkflowHistory(req.Namespace, limit, offset)
	if err != nil {
		return nil, err
	}
	allowedItems := make([]wfv1.Workflow, 0)
	// TODO this loop Hibernates 1+N and is likely to very slow for large requests, needs testing
	for _, wf := range allItems {
		allowed, err := auth.CanI(ctx, "get", "workflow", req.Namespace, wf.Name)
		if err != nil {
			return nil, err
		}
		if allowed {
			allowedItems = append(allowedItems, wf)
		}
	}
	meta := metav1.ListMeta{}
	if len(allowedItems) >= limit {
		meta.Continue = fmt.Sprintf("%v", offset+limit)
	}
	return &wfv1.WorkflowList{ListMeta: meta, Items: allowedItems}, nil
}

func (w *workflowHistoryServer) GetWorkflowHistory(ctx context.Context, req *WorkflowHistoryGetRequest) (*wfv1.Workflow, error) {
	wf, err := w.repo.GetWorkflowHistory(req.Namespace, req.Uid)
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

func (w *workflowHistoryServer) ResubmitWorkflowHistory(ctx context.Context, req *WorkflowHistoryUpdateRequest) (*wfv1.Workflow, error) {
	wf, err := w.GetWorkflowHistory(ctx, &WorkflowHistoryGetRequest{Namespace: req.Namespace, Uid: req.Uid})
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

func (w *workflowHistoryServer) DeleteWorkflowHistory(ctx context.Context, req *WorkflowHistoryDeleteRequest) (*WorkflowHistoryDeleteResponse, error) {
	wf, err := w.GetWorkflowHistory(ctx, &WorkflowHistoryGetRequest{Namespace: req.Namespace, Uid: req.Uid})
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
	err = w.repo.DeleteWorkflowHistory(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}
	return &WorkflowHistoryDeleteResponse{}, nil
}
