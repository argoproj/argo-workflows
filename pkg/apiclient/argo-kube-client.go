package apiclient

import (
	"context"
	"fmt"

	events "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	clusterworkflowtmplserver "github.com/argoproj/argo-workflows/v3/server/clusterworkflowtemplate"
	cronworkflowserver "github.com/argoproj/argo-workflows/v3/server/cronworkflow"
	syncserver "github.com/argoproj/argo-workflows/v3/server/sync"
	"github.com/argoproj/argo-workflows/v3/server/types"
	workflowserver "github.com/argoproj/argo-workflows/v3/server/workflow"
	"github.com/argoproj/argo-workflows/v3/server/workflow/store"
	workflowtemplateserver "github.com/argoproj/argo-workflows/v3/server/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/util/help"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	rbacutil "github.com/argoproj/argo-workflows/v3/util/rbac"
)

var (
	argoKubeOffloadNodeStatusRepo = sqldb.ExplosiveOffloadNodeStatusRepo
	ErrNoArgoServer               = fmt.Errorf("this is impossible if you are not using the Argo Server, see %s", help.CLI())
)

type ArgoKubeOpts struct {
	// Closing caching channel will stop caching informers
	CachingCloseCh chan struct{}

	// Whether to cache Workflows
	// This improves performance of reading Workflows, but it increases memory usage and startup time
	//
	// Workflow caching uses in-memory SQLite DB and it provides full capabilities
	CacheWorkflows bool

	// Whether to cache WorkflowTemplates
	// This improves performance of reading WorkflowTemplates, but it increases memory usage and startup time
	// It is especially visible during validating templates with many references,
	//
	// Note that templates caching currently uses informers, so not all template
	// get/list can use it, since informer has limited capabilities (such as filtering)
	CacheWorkflowTemplates bool

	// Whether to cache ClusterWorkflowTemplates
	// This improves performance of reading ClusterWorkflowTemplates, but it increases memory usage and startup time
	// It is especially visible during validating templates with many references,
	//
	// Note that templates caching currently uses informers, so not all template
	// get/list can use it, since informer has limited capabilities (such as filtering)
	CacheClusterWorkflowTemplates bool
}

type argoKubeClient struct {
	opts              ArgoKubeOpts
	instanceIDService instanceid.Service
	wfClient          workflow.Interface
	wfTmplStore       types.WorkflowTemplateStore
	cwfTmplStore      types.ClusterWorkflowTemplateStore
	wfLister          store.WorkflowLister
	wfStore           store.WorkflowStore
	namespace         string
	kubeClient        *kubernetes.Clientset
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
	eventInterface, err := events.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	clients := &types.Clients{
		Dynamic:    dynamicClient,
		Events:     eventInterface,
		Kubernetes: kubeClient,
		Workflow:   wfClient,
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
		namespace:         namespace,
		kubeClient:        kubeClient,
	}
	err = client.startStores(ctx, restConfig)
	if err != nil {
		return nil, nil, err
	}

	return ctx, client, nil
}

func (a *argoKubeClient) startStores(ctx context.Context, restConfig *restclient.Config) error {
	if a.opts.CacheWorkflows {
		wfStore, err := store.NewSQLiteStore(a.instanceIDService)
		if err != nil {
			return err
		}
		a.wfStore = wfStore
		a.wfLister = wfStore
	} else {
		a.wfLister = store.NewKubeLister(a.wfClient)
	}

	if a.opts.CacheWorkflowTemplates {
		wftmplInformer, err := workflowtemplateserver.NewInformer(restConfig, a.namespace)
		if err != nil {
			return err
		}
		wftmplInformer.Run(ctx, a.opts.CachingCloseCh)
		a.wfTmplStore = wftmplInformer
	} else {
		a.wfTmplStore = workflowtemplateserver.NewWorkflowTemplateClientStore()
	}

	if rbacutil.HasAccessToClusterWorkflowTemplates(ctx, a.kubeClient, a.namespace) {
		if a.opts.CacheClusterWorkflowTemplates {
			cwftmplInformer, err := clusterworkflowtmplserver.NewInformer(restConfig)
			if err != nil {
				return err
			}
			cwftmplInformer.Run(ctx, a.opts.CachingCloseCh)
			a.cwfTmplStore = cwftmplInformer
		} else {
			a.cwfTmplStore = clusterworkflowtmplserver.NewClusterWorkflowTemplateClientStore()
		}
	} else {
		a.cwfTmplStore = clusterworkflowtmplserver.NewNullClusterWorkflowTemplate()
	}

	return nil
}

func (a *argoKubeClient) NewWorkflowServiceClient(ctx context.Context) workflowpkg.WorkflowServiceClient {
	wfArchive := sqldb.NullWorkflowArchive
	wfServer := workflowserver.NewWorkflowServer(ctx, a.instanceIDService, argoKubeOffloadNodeStatusRepo, wfArchive, a.wfClient, a.wfLister, a.wfStore, a.wfTmplStore, a.cwfTmplStore, nil, &a.namespace)
	go wfServer.Run(a.opts.CachingCloseCh)
	return &errorTranslatingWorkflowServiceClient{&argoKubeWorkflowServiceClient{wfServer}}
}

func (a *argoKubeClient) NewCronWorkflowServiceClient() (cronworkflow.CronWorkflowServiceClient, error) {
	return &errorTranslatingCronWorkflowServiceClient{&argoKubeCronWorkflowServiceClient{cronworkflowserver.NewCronWorkflowServer(a.instanceIDService, a.wfTmplStore, a.cwfTmplStore, nil)}}, nil
}

func (a *argoKubeClient) NewWorkflowTemplateServiceClient() (workflowtemplate.WorkflowTemplateServiceClient, error) {
	return &errorTranslatingWorkflowTemplateServiceClient{&argoKubeWorkflowTemplateServiceClient{workflowtemplateserver.NewWorkflowTemplateServer(a.instanceIDService, a.wfTmplStore, a.cwfTmplStore)}}, nil
}

func (a *argoKubeClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, ErrNoArgoServer
}

func (a *argoKubeClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return nil, ErrNoArgoServer
}

func (a *argoKubeClient) NewClusterWorkflowTemplateServiceClient() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error) {
	return &errorTranslatingWorkflowClusterTemplateServiceClient{&argoKubeWorkflowClusterTemplateServiceClient{clusterworkflowtmplserver.NewClusterWorkflowTemplateServer(a.instanceIDService, a.cwfTmplStore, nil)}}, nil
}

func (a *argoKubeClient) NewSyncServiceClient(ctx context.Context) (syncpkg.SyncServiceClient, error) {
	return &errorTranslatingArgoKubeSyncServiceClient{&argoKubeSyncServiceClient{syncserver.NewSyncServer(ctx, nil, "", nil)}}, nil
}
