package apiclient

import (
	"context"
	"fmt"

	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
)

var OfflineErr = fmt.Errorf("not supported when you are in offline mode")

type offlineClient struct {
}

func newOfflineClient() (context.Context, Client, error) {
	return context.Background(), &offlineClient{}, nil
}

func (o offlineClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, OfflineErr
}

func (o offlineClient) NewWorkflowServiceClient() (workflowpkg.WorkflowServiceClient, error) {
	return &offlineWorkflowServiceClient{}, nil
}

func (o offlineClient) NewCronWorkflowServiceClient() (cronworkflowpkg.CronWorkflowServiceClient, error) {
	return nil, OfflineErr
}

func (o offlineClient) NewWorkflowTemplateServiceClient() (workflowtemplatepkg.WorkflowTemplateServiceClient, error) {
	return nil, OfflineErr
}

func (o offlineClient) NewClusterWorkflowTemplateServiceClient() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error) {
	return nil, OfflineErr
}

func (o offlineClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return nil, OfflineErr
}
