package apiclient

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/auth"
	clusterworkflowtmplserver "github.com/argoproj/argo/server/clusterworkflowtemplate"
	cronworkflowserver "github.com/argoproj/argo/server/cronworkflow"
	workflowserver "github.com/argoproj/argo/server/workflow"
	workflowtemplateserver "github.com/argoproj/argo/server/workflowtemplate"
	"github.com/argoproj/argo/util/help"
)

var argoKubeOffloadNodeStatusRepo = sqldb.ExplosiveOffloadNodeStatusRepo

type argoKubeClient struct {
	instanceID string
}

func newArgoKubeClient(clientConfig clientcmd.ClientConfig, instanceID string) (context.Context, Client, error) {
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
	return ctx, &argoKubeClient{instanceID}, nil
}

func (a *argoKubeClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return &argoKubeWorkflowServiceClient{workflowserver.NewWorkflowServer(a.instanceID, argoKubeOffloadNodeStatusRepo)}
}

func (a *argoKubeClient) NewCronWorkflowServiceClient() cronworkflow.CronWorkflowServiceClient {
	return &argoKubeCronWorkflowServiceClient{cronworkflowserver.NewCronWorkflowServer(a.instanceID)}
}

func (a *argoKubeClient) NewWorkflowTemplateServiceClient() workflowtemplate.WorkflowTemplateServiceClient {
	return &argoKubeWorkflowTemplateServiceClient{workflowtemplateserver.NewWorkflowTemplateServer(a.instanceID)}
}

func (a *argoKubeClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, fmt.Errorf("it is impossible to interact with the workflow archive if you are not using the Argo Server, see " + help.CLI)
}

func (a *argoKubeClient) NewClusterWorkflowTemplateServiceClient() clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient {
	return &argoKubeWorkflowClusterTemplateServiceClient{clusterworkflowtmplserver.NewClusterWorkflowTemplateServer(a.instanceID)}
}
