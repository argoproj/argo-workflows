package workflowhistory

import (
	"context"

	"k8s.io/client-go/kubernetes"

	commonserver "github.com/argoproj/argo/cmd/server/common"
	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/config"
)

type workflowHistoryServer struct {
	*commonserver.Server
	repo sqldb.WorkflowHistoryRepository
}

func NewWorkflowHistoryServer(namespace string, wfClientset versioned.Interface, kubeClientset kubernetes.Interface, enableClientAuth bool, persistConfig *config.PersistConfig) (*workflowHistoryServer, error) {
	database, _, err := sqldb.CreateDBSession(kubeClientset, namespace, persistConfig)
	if err != nil {
		return nil, err
	}
	return &workflowHistoryServer{
		Server: commonserver.NewServer(enableClientAuth, namespace, wfClientset, kubeClientset),
		repo:   sqldb.NewWorkflowHistoryRepository(database),
	}, err
}

func (w workflowHistoryServer) ListWorkflowHistory(ctx context.Context, req *WorkflowHistoryListRequest) (*wfv1.WorkflowList, error) {
	history, err := w.repo.ListWorkflowHistory()
	if err != nil {
		return nil, err
	}
	return &wfv1.WorkflowList{Items: history}, nil
}
