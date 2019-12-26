package workflowhistory

import (
	"context"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
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
	/*
		var repo sqldb.WorkflowHistoryRepository
		if persistConfig != nil {
			database, _, err := sqldb.CreateDBSession(kubeClientset, namespace, persistConfig)
			if err != nil {
				return nil, err
			}
			repo = sqldb.NewWorkflowHistoryRepository(database)
		} else {
			repo = sqldb.NullWorkflowHistoryRepository
		}
	*/
	return &workflowHistoryServer{repo: repo, kubeClientset: kubeClientset}, nil
}

func (w workflowHistoryServer) ListWorkflowHistory(ctx context.Context, req *WorkflowHistoryListRequest) (*wfv1.WorkflowList, error) {
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
		review, err := w.kubeClientset.AuthorizationV1().SelfSubjectAccessReviews().Create(&authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: item.Namespace,
					Verb:      "get",
					Group:     item.GroupVersionKind().Group,
					Version:   item.GroupVersionKind().Version,
					Resource:  "workflows",
					Name:      item.Name,
				},
			},
		})
		if err != nil {
			return nil, err
		}
		if review.Status.Allowed {
			allowedItems = append(allowedItems, item)
		} else {
			log.WithFields(log.Fields{"review": review}).Warn("Access denied")
		}
	}
	meta := metav1.ListMeta{}
	if len(allowedItems) >= limit {
		meta.Continue = fmt.Sprintf("%v", offset+limit)
	}
	return &wfv1.WorkflowList{ListMeta: meta, Items: allowedItems}, nil
}
