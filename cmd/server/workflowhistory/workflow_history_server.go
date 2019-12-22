package workflowhistory

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/config"
)

type workflowHistoryServer struct {
	repo sqldb.WorkflowHistoryRepository
}

func NewWorkflowHistoryServer(namespace string, kubeClientset kubernetes.Interface, persistConfig *config.PersistConfig) (*workflowHistoryServer, error) {
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
	return &workflowHistoryServer{repo: repo}, nil
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
	history, err := w.repo.ListWorkflowHistory(limit, offset)
	if err != nil {
		return nil, err
	}
	meta := metav1.ListMeta{}
	if len(history) >= limit {
		meta.Continue = fmt.Sprintf("%v", offset+limit)
	}
	return &wfv1.WorkflowList{ListMeta: meta, Items: history}, nil
}
