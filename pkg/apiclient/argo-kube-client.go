package apiclient

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
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
	"github.com/argoproj/argo/util/instanceid"
)

var argoKubeOffloadNodeStatusRepo = sqldb.ExplosiveOffloadNodeStatusRepo
var NoArgoServerErr = fmt.Errorf("this is impossible if you are not using the Argo Server, see " + help.CLI)

type argoKubeClient struct {
	instanceIDService instanceid.Service
}

func newArgoKubeClient(clientConfig clientcmd.ClientConfig, instanceIDService instanceid.Service) (context.Context, Client, error) {
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
	gatekeeper, err := auth.NewGatekeeper(auth.Modes{auth.Server: true}, wfClient, kubeClient, restConfig, nil, auth.DefaultClientForAuthorization, "unused")
	if err != nil {
		return nil, nil, err
	}
	ctx, err := gatekeeper.Context(context.Background())
	if err != nil {
		return nil, nil, err
	}
	return ctx, &argoKubeClient{instanceIDService}, nil
}

func (a *argoKubeClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return &errorTranslatingWorkflowServiceClient{&argoKubeWorkflowServiceClient{workflowserver.NewWorkflowServer(a.instanceIDService, argoKubeOffloadNodeStatusRepo)}}
}

func (a *argoKubeClient) NewCronWorkflowServiceClient() cronworkflow.CronWorkflowServiceClient {
	return &errorTranslatingCronWorkflowServiceClient{&argoKubeCronWorkflowServiceClient{cronworkflowserver.NewCronWorkflowServer(a.instanceIDService)}}
}

func (a *argoKubeClient) NewWorkflowTemplateServiceClient() workflowtemplate.WorkflowTemplateServiceClient {
	return &errorTranslatingWorkflowTemplateServiceClient{&argoKubeWorkflowTemplateServiceClient{workflowtemplateserver.NewWorkflowTemplateServer(a.instanceIDService)}}
}

func (a *argoKubeClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, NoArgoServerErr
}

func (a *argoKubeClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return nil, NoArgoServerErr
}

func (a *argoKubeClient) NewClusterWorkflowTemplateServiceClient() clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient {
	return &errorTranslatingWorkflowClusterTemplateServiceClient{&argoKubeWorkflowClusterTemplateServiceClient{clusterworkflowtmplserver.NewClusterWorkflowTemplateServer(a.instanceIDService)}}
}
