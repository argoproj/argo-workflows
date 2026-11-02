package apiclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/http1"
	infopkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/info"
	syncpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/sync"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
)

type httpClient http1.Facade

var _ Client = &httpClient{}

func (h httpClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return http1.ArchivedWorkflowsServiceClient(h), nil
}

func (h httpClient) NewWorkflowServiceClient(_ context.Context) workflowpkg.WorkflowServiceClient {
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

func (h httpClient) NewSyncServiceClient(_ context.Context) (syncpkg.SyncServiceClient, error) {
	return http1.SyncServiceClient(h), nil
}

func newHTTP1Client(ctx context.Context, opts ArgoServerOpts, auth string, proxy func(*http.Request) (*url.URL, error)) (context.Context, Client, error) {
	facade, err := http1.NewFacade(http1.FacadeConfig{
		BaseURL:            opts.GetURL(),
		Authorization:      auth,
		InsecureSkipVerify: opts.InsecureSkipVerify,
		Headers:            opts.Headers,
		HTTPClient:         opts.HTTP1Client,
		Proxy:              proxy,
		ClientCert:         opts.ClientCert,
		ClientKey:          opts.ClientKey,
		CACert:             opts.CACert,
	})
	if err != nil {
		return nil, nil, err
	}
	return ctx, httpClient(facade), nil
}
