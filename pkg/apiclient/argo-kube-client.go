package apiclient

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/auth"
	cronworkflowserver "github.com/argoproj/argo/server/cronworkflow"
	workflowserver "github.com/argoproj/argo/server/workflow"
	workflowtemplateserver "github.com/argoproj/argo/server/workflowtemplate"
	"github.com/argoproj/argo/util/help"
)

var argoKubeOffloadNodeStatusRepo = sqldb.ExplosiveOffloadNodeStatusRepo

type argoKubeClient struct {
}

func newArgoKubeClient(clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	wfClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	gatekeeper := auth.NewGatekeeper(auth.Server, wfClient, kubeClient, restConfig)
	ctx, err := gatekeeper.Context(context.Background())
	if err != nil {
		return nil, nil, err
	}
	return ctx, &argoKubeClient{}, nil
}

func (a *argoKubeClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return &argoKubeWorkflowServiceClient{workflowserver.NewWorkflowServer(argoKubeOffloadNodeStatusRepo)}
}

func (a *argoKubeClient) NewCronWorkflowServiceClient() cronworkflow.CronWorkflowServiceClient {
	return &argoKubeCronWorkflowServiceClient{cronworkflowserver.NewCronWorkflowServer()}
}
func (a *argoKubeClient) NewWorkflowTemplateServiceClient() workflowtemplate.WorkflowTemplateServiceClient {
	return &argoKubeWorkflowTemplateServiceClient{workflowtemplateserver.NewWorkflowTemplateServer()}
}

func (a *argoKubeClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, fmt.Errorf("it is impossible to interact with the workflow archive if you are not using the Argo Server, see " + help.CLI)
}
