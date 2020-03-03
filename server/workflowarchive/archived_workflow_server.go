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
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo/v2/persist/sqldb"
	workflowarchivepkg "github.com/argoproj/argo/v2/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo/v2/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/v2/server/auth"
)

type archivedWorkflowServer struct {
	wfArchive sqldb.WorkflowArchive
}

func NewWorkflowArchiveServer(wfArchive sqldb.WorkflowArchive) workflowarchivepkg.ArchivedWorkflowServiceServer {
	return &archivedWorkflowServer{wfArchive: wfArchive}
}

func (w *archivedWorkflowServer) ListArchivedWorkflows(ctx context.Context, req *workflowarchivepkg.ListArchivedWorkflowsRequest) (*wfv1.WorkflowList, error) {
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

	requirements, err := labels.ParseToRequirements(options.LabelSelector)
	if err != nil {
		return nil, err
	}

	items := make(wfv1.Workflows, 0)
	authorizer := auth.NewAuthorizer(ctx)
	hasMore := true
	// keep trying until we have enough
	for len(items) < limit {
		moreItems, err := w.wfArchive.ListWorkflows(namespace, requirements, limit+1, offset)
		if err != nil {
			return nil, err
		}
		for index, wf := range moreItems {
			if index == limit {
				break
			}
			// TODO second pass filtering?
			allowed, err := authorizer.CanI("get", workflow.WorkflowPlural, wf.Namespace, wf.Name)
			if err != nil {
				return nil, err
			}
			if allowed {
				items = append(items, wf)
			}
		}
		if len(moreItems) < limit+1 {
			hasMore = false
			break
		}
		offset = offset + limit
	}
	meta := metav1.ListMeta{}
	if hasMore {
		meta.Continue = fmt.Sprintf("%v", offset)
	}
	sort.Sort(items)
	return &wfv1.WorkflowList{ListMeta: meta, Items: items}, nil
}

func (w *archivedWorkflowServer) GetArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.GetArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wf, err := w.wfArchive.GetWorkflow(req.Uid)
	if err != nil {
		return nil, err
	}
	if wf == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	allowed, err := auth.CanI(ctx, "get", workflow.WorkflowPlural, wf.Namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return wf, err
}

func (w *archivedWorkflowServer) DeleteArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.DeleteArchivedWorkflowRequest) (*workflowarchivepkg.ArchivedWorkflowDeletedResponse, error) {
	wf, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid})
	if err != nil {
		return nil, err
	}
	allowed, err := auth.CanI(ctx, "delete", workflow.WorkflowPlural, wf.Namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	err = w.wfArchive.DeleteWorkflow(req.Uid)
	if err != nil {
		return nil, err
	}
	return &workflowarchivepkg.ArchivedWorkflowDeletedResponse{}, nil
}
