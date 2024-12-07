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
	"github.com/argoproj/argo-workflows/v3/server/workflow/store"
	workflowstore "github.com/argoproj/argo-workflows/v3/server/workflow/store"
	workflowtemplateserver "github.com/argoproj/argo-workflows/v3/server/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/util/help"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
)

var (
	argoKubeOffloadNodeStatusRepo = sqldb.ExplosiveOffloadNodeStatusRepo
	NoArgoServerErr               = fmt.Errorf("this is impossible if you are not using the Argo Server, see %s", help.CLI())
)

type ArgoKubeOpts struct {
	// Closing caching channel will stop caching informers
	CachingCloseCh chan struct{}

	// Whether to cache WorkflowTemplates, ClusterWorkflowTemplates and Workflows
	// This improves performance of reading
	// It is especially visible during validating templates,
	//
	// Note that templates caching currently uses informers, so not all template
	// get/list can use it, since informer has limited capabilities (such as filtering)
	//
	// Workflow caching uses in-memory SQLite DB and it provides full capabilities
	UseCaching bool
}

type argoKubeClient struct {
	opts              ArgoKubeOpts
	instanceIDService instanceid.Service
	wfClient          workflow.Interface
	wfTmplStore       types.WorkflowTemplateStore
	cwfTmplStore      types.ClusterWorkflowTemplateStore
	wfLister          workflowstore.WorkflowLister
	wfStore           workflowstore.WorkflowStore
}

var _ Client = &argoKubeClient{}

func newArgoKubeClient(ctx context.Context, opts ArgoKubeOpts, clientConfig clientcmd.ClientConfig, instanceIDService instanceid.Service) (context.Context, Client, error) {
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
	namespace, _, err := clientConfig.Namespace()
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

	client := &argoKubeClient{
		opts:              opts,
		instanceIDService: instanceIDService,
		wfClient:          wfClient,
	}
	err = client.startStores(restConfig, namespace)
	if err != nil {
		return nil, nil, err
	}

	return ctx, client, nil
}

func (a *argoKubeClient) startStores(restConfig *restclient.Config, namespace string) error {
	if a.opts.UseCaching {
		wftmplInformer, err := workflowtemplateserver.NewInformer(restConfig, namespace)
		if err != nil {
			return err
		}
		cwftmplInformer, err := clusterworkflowtmplserver.NewInformer(restConfig)
		if err != nil {
			return err
		}
		wfStore, err := store.NewSQLiteStore(a.instanceIDService)
		if err != nil {
			return err
		}
		wftmplInformer.Run(a.opts.CachingCloseCh)
		cwftmplInformer.Run(a.opts.CachingCloseCh)
		a.wfStore = wfStore
		a.wfLister = wfStore
		a.wfTmplStore = wftmplInformer
		a.cwfTmplStore = cwftmplInformer
	} else {
		a.wfLister = store.NewKubeLister(a.wfClient)
		a.wfTmplStore = workflowtemplateserver.NewWorkflowTemplateClientStore()
		a.cwfTmplStore = clusterworkflowtmplserver.NewClusterWorkflowTemplateClientStore()
	}
	return nil
}

func (a *argoKubeClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	wfArchive := sqldb.NullWorkflowArchive
	return &errorTranslatingWorkflowServiceClient{&argoKubeWorkflowServiceClient{workflowserver.NewWorkflowServer(a.instanceIDService, argoKubeOffloadNodeStatusRepo, wfArchive, a.wfClient, a.wfLister, a.wfStore, a.wfTmplStore, a.cwfTmplStore, nil)}}
}

func (a *argoKubeClient) NewCronWorkflowServiceClient() (cronworkflow.CronWorkflowServiceClient, error) {
	return &errorTranslatingCronWorkflowServiceClient{&argoKubeCronWorkflowServiceClient{cronworkflowserver.NewCronWorkflowServer(a.instanceIDService, a.wfTmplStore, a.cwfTmplStore)}}, nil
}

func (a *argoKubeClient) NewWorkflowTemplateServiceClient() (workflowtemplate.WorkflowTemplateServiceClient, error) {
	return &errorTranslatingWorkflowTemplateServiceClient{&argoKubeWorkflowTemplateServiceClient{workflowtemplateserver.NewWorkflowTemplateServer(a.instanceIDService, a.wfTmplStore, a.cwfTmplStore)}}, nil
}

func (a *argoKubeClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, NoArgoServerErr
}

func (a *argoKubeClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return nil, NoArgoServerErr
}

func (a *argoKubeClient) NewClusterWorkflowTemplateServiceClient() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error) {
	return &errorTranslatingWorkflowClusterTemplateServiceClient{&argoKubeWorkflowClusterTemplateServiceClient{clusterworkflowtmplserver.NewClusterWorkflowTemplateServer(a.instanceIDService, a.cwfTmplStore)}}, nil
}
