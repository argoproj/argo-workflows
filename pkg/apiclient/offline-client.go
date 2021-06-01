package apiclient

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
)

type offlineClient struct{}

var NotImplError error = fmt.Errorf("Not implemented for offline client, only valid for kind '--kinds=workflows'")

var _ Client = &offlineClient{}

func newOfflineClient() (context.Context, Client, error) {
	return context.Background(), &offlineClient{}, nil
}

func (a *offlineClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return &errorTranslatingWorkflowServiceClient{OfflineWorkflowServiceClient{}}
}

func (a *offlineClient) NewCronWorkflowServiceClient() (cronworkflow.CronWorkflowServiceClient, error) {
	return nil, NotImplError
}

func (a *offlineClient) NewWorkflowTemplateServiceClient() (workflowtemplate.WorkflowTemplateServiceClient, error) {
	return nil, NotImplError
}

func (a *offlineClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, NotImplError
}

func (a *offlineClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return nil, NotImplError
}

func (a *offlineClient) NewClusterWorkflowTemplateServiceClient() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error) {
	return nil, NotImplError
}
