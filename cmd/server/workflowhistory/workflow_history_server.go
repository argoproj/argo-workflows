package workflowhistory

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type workflowHistoryServer struct {
	kubeClientset kubernetes.Interface
	repo          sqldb.WorkflowHistoryRepository
}

func NewWorkflowHistoryServer(kubeClientset kubernetes.Interface, repo sqldb.WorkflowHistoryRepository) (*workflowHistoryServer, error) {
	return &workflowHistoryServer{repo: repo, kubeClientset: kubeClientset}, nil
}

func (w *workflowHistoryServer) ListWorkflowHistory(_ context.Context, req *WorkflowHistoryListRequest) (*wfv1.WorkflowList, error) {
	options := req.ListOptions
	if options == nil {
		options = &metav1.ListOptions{Limit: 100}
	}
	if options.Continue == "" {
		options.Continue = "0"
	}
	limit := int(options.Limit)
	offset, err := strconv.Atoi(options.Continue)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "listOptions.continue must be int")
	}
	allItems, err := w.repo.ListWorkflowHistory(limit, offset)
	if err != nil {
		return nil, err
	}
	allowedItems := make([]wfv1.Workflow, 0)
	// TODO this loop Hibernates 1+N and is likely to very slow for large requests, needs testing
	for _, item := range allItems {
		allowed, err := w.isAllowed(&item)
		if err != nil {
			return nil, err
		}
		if allowed {
			allowedItems = append(allowedItems, item)
		}
	}
	meta := metav1.ListMeta{}
	if len(allowedItems) >= limit {
		meta.Continue = fmt.Sprintf("%v", offset+limit)
	}
	return &wfv1.WorkflowList{ListMeta: meta, Items: allowedItems}, nil
}

func (w *workflowHistoryServer) isAllowed(wf *wfv1.Workflow) (bool, error) {
	review, err := w.kubeClientset.AuthorizationV1().SelfSubjectAccessReviews().Create(&authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: wf.Namespace,
				Verb:      "get",
				Group:     wf.GroupVersionKind().Group,
				Version:   wf.GroupVersionKind().Version,
				Resource:  "workflows",
				Name:      wf.Name,
			},
		},
	})
	if err != nil {
		return false, err
	}
	return review.Status.Allowed, nil
}

func (w *workflowHistoryServer) GetWorkflowHistory(_ context.Context, req *WorkflowHistoryGetRequest) (*wfv1.Workflow, error) {
	wf, err := w.repo.GetWorkflowHistory(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}
	if wf == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	allowed, err := w.isAllowed(wf)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return wf, err
}
