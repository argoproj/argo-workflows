package apiclient

import (
	"context"

	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apiclient/http"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
)

type httpClient http.Facade

func (h httpClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return http.ArchivedWorkflowsServiceClient(h), nil
}

func (h httpClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return http.WorkflowServiceClient(h)
}

func (h httpClient) NewCronWorkflowServiceClient() cronworkflowpkg.CronWorkflowServiceClient {
	return http.CronWorkflowServiceClient(h)
}

func (h httpClient) NewWorkflowTemplateServiceClient() workflowtemplatepkg.WorkflowTemplateServiceClient {
	return http.WorkflowTemplateServiceClient(h)
}

func (h httpClient) NewClusterWorkflowTemplateServiceClient() clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient {
	return http.ClusterWorkflowTemplateServiceClient(h)
}

func (h httpClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return http.InfoServiceClient(h), nil
}

func newHTTPClient(baseUrl string, auth string) (context.Context, Client, error) {
	return context.Background(), httpClient(http.NewFacade(baseUrl, auth)), nil
}
