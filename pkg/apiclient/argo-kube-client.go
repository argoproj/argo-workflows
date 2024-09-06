package apiclient

import (
	"context"
	"fmt"

	eventsource "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v3/server/workflow/store"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	clusterworkflowtmplserver "github.com/argoproj/argo-workflows/v3/server/clusterworkflowtemplate"
	cronworkflowserver "github.com/argoproj/argo-workflows/v3/server/cronworkflow"
	"github.com/argoproj/argo-workflows/v3/server/types"
	workflowserver "github.com/argoproj/argo-workflows/v3/server/workflow"
	workflowtemplateserver "github.com/argoproj/argo-workflows/v3/server/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/util/help"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
)

var (
	argoKubeOffloadNodeStatusRepo = sqldb.ExplosiveOffloadNodeStatusRepo
	NoArgoServerErr               = fmt.Errorf("this is impossible if you are not using the Argo Server, see " + help.CLI())
)

type argoKubeClient struct {
	instanceIDService instanceid.Service
	wfClient          workflow.Interface
}

var _ Client = &argoKubeClient{}

func newArgoKubeClient(ctx context.Context, clientConfig clientcmd.ClientConfig, instanceIDService instanceid.Service) (context.Context, Client, error) {
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	version := argo.GetVersion()
	restConfig = restclient.AddUserAgent(restConfig, fmt.Sprintf("argo-workflows/%s argo-api-client", version.Version))
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failure to create dynamic client: %w", err)
	}
	wfClient, err := workflow.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	eventSourceInterface, err := eventsource.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	sensorInterface, err := sensor.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	clients := &types.Clients{
		Dynamic:     dynamicClient,
		EventSource: eventSourceInterface,
		Kubernetes:  kubeClient,
		Sensor:      sensorInterface,
		Workflow:    wfClient,
	}
	gatekeeper, err := auth.NewGatekeeper(auth.Modes{auth.Server: true}, clients, restConfig, nil, auth.DefaultClientForAuthorization, "unused", "unused", false, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, err = gatekeeper.Context(ctx)
	if err != nil {
		return nil, nil, err
	}
	return ctx, &argoKubeClient{instanceIDService, wfClient}, nil
}

func (a *argoKubeClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	wfArchive := sqldb.NullWorkflowArchive
	wfLister := store.NewKubeLister(a.wfClient)
	return &errorTranslatingWorkflowServiceClient{&argoKubeWorkflowServiceClient{workflowserver.NewWorkflowServer(a.instanceIDService, argoKubeOffloadNodeStatusRepo, wfArchive, a.wfClient, wfLister, nil, nil)}}
}

func (a *argoKubeClient) NewCronWorkflowServiceClient() (cronworkflow.CronWorkflowServiceClient, error) {
	return &errorTranslatingCronWorkflowServiceClient{&argoKubeCronWorkflowServiceClient{cronworkflowserver.NewCronWorkflowServer(a.instanceIDService)}}, nil
}

func (a *argoKubeClient) NewWorkflowTemplateServiceClient() (workflowtemplate.WorkflowTemplateServiceClient, error) {
	return &errorTranslatingWorkflowTemplateServiceClient{&argoKubeWorkflowTemplateServiceClient{workflowtemplateserver.NewWorkflowTemplateServer(a.instanceIDService)}}, nil
}

func (a *argoKubeClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, NoArgoServerErr
}

func (a *argoKubeClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return nil, NoArgoServerErr
}

func (a *argoKubeClient) NewClusterWorkflowTemplateServiceClient() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error) {
	return &errorTranslatingWorkflowClusterTemplateServiceClient{&argoKubeWorkflowClusterTemplateServiceClient{clusterworkflowtmplserver.NewClusterWorkflowTemplateServer(a.instanceIDService)}}, nil
}
