package apiclient

import (
	"context"
	"net/http"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/http1"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
)

type httpClient http1.Facade

var _ Client = &httpClient{}

func (h httpClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return http1.ArchivedWorkflowsServiceClient(h), nil
}

func (h httpClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return http1.WorkflowServiceClient(h)
}

func (h httpClient) NewCronWorkflowServiceClient() (cronworkflowpkg.CronWorkflowServiceClient, error) {
	return http1.CronWorkflowServiceClient(h), nil
}

func (h httpClient) NewWorkflowTemplateServiceClient() (workflowtemplatepkg.WorkflowTemplateServiceClient, error) {
	return http1.WorkflowTemplateServiceClient(h), nil
}

func (h httpClient) NewClusterWorkflowTemplateServiceClient() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error) {
	return http1.ClusterWorkflowTemplateServiceClient(h), nil
}

func (h httpClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return http1.InfoServiceClient(h), nil
}

func newHTTP1Client(baseURL string, auth string, insecureSkipVerify bool, headers []string, customHTTPClient *http.Client) (context.Context, Client, error) {
	return logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())), httpClient(http1.NewFacade(baseURL, auth, insecureSkipVerify, headers, customHTTPClient)), nil
}
