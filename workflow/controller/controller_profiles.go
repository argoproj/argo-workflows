package controller

import (
	"context"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
)

func (wfc *WorkflowController) loadProfiles(ctx context.Context, kubernetesClient kubernetes.Interface) error {
	list, err := kubernetesClient.CoreV1().Secrets(wfc.namespace).List(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyCluster})
	if err != nil {
		return err
	}
	for _, secret := range list.Items {
		if err := wfc.loadProfile(&secret); err != nil {
			return err
		}
	}
	return nil
}

func (wfc *WorkflowController) loadPrimaryProfile(restConfig *rest.Config, kubernetesClient kubernetes.Interface, workflowClient wfclientset.Interface, metadataClient metadata.Interface) {
	log.Info("Loading primary profile")
	wfc.profiles[wfc.primaryProfileKey()] = &profile{
		policy: policy{
			workflowNamespace: wfc.GetManagedNamespace(),
			cluster:           common.LocalCluster,
			namespace:         wfc.GetManagedNamespace(),
		},
		restConfig:         restConfig,
		kubernetesClient:   kubernetesClient,
		workflowClient:     workflowClient,
		metadataClient:     metadataClient,
		podInformer:        wfc.newPodInformer(kubernetesClient, common.LocalCluster, wfc.GetManagedNamespace()),
		podGCInformer:      wfc.newPodGCInformer(metadataClient, common.LocalCluster, wfc.GetManagedNamespace()),
		taskResultInformer: wfc.newWorkflowTaskResultInformer(workflowClient, common.LocalCluster, wfc.GetManagedNamespace()),
	}
}

func (wfc *WorkflowController) loadProfile(secret *apiv1.Secret) error {

	key := profileKey(wfc.clusterOf(secret))

	log.WithField("key", key).Info("Loading profile")

	clientConfig, err := clientcmd.Load(secret.Data["kubeconfig"])
	if err != nil {
		return err
	}

	config, err := clientcmd.NewNonInteractiveClientConfig(*clientConfig, clientConfig.CurrentContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()).ClientConfig()
	if err != nil {
		return err
	}

	logs.AddK8SLogTransportWrapper(config)
	metrics.AddMetricsTransportWrapper(config)

	kubernetesClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	namespace := common.Namespace(secret)

	workflowNamespace, cluster := common.ClusterWorkflowNamespace(secret, wfc.primaryCluster())
	log.
		WithField("currentContext", clientConfig.CurrentContext).
		WithField("configHost", config.Host).
		WithField("secretNamespace", secret.Namespace).
		WithField("secretName", secret.Name).
		WithField("workflowNamespace", workflowNamespace).
		WithField("cluster", cluster).
		WithField("namespace", namespace).
		Info("Profile configuration")

	workflowClient, err := wfclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		return err
	}
	p := &profile{
		policy: policy{
			workflowNamespace: workflowNamespace,
			cluster:           cluster,
			namespace:         namespace,
		},
		restConfig:         config,
		kubernetesClient:   kubernetesClient,
		workflowClient:     workflowClient,
		metadataClient:     metadataClient,
		podInformer:        wfc.newPodInformer(kubernetesClient, cluster, namespace),
		podGCInformer:      wfc.newPodGCInformer(metadataClient, cluster, namespace),
		taskResultInformer: wfc.newWorkflowTaskResultInformer(workflowClient, cluster, namespace),
		done:               make(chan struct{}),
	}
	wfc.profiles[key] = p
	return nil
}

func (wfc *WorkflowController) primaryCluster() string {
	return wfc.Config.Cluster
}

func (wfc *WorkflowController) primaryProfileKey() profileKey {
	return profileKey(wfc.Config.Cluster)
}

func (wfc *WorkflowController) primaryProfile() *profile {
	return wfc.profiles[wfc.primaryProfileKey()]
}

func (wfc *WorkflowController) profile(workflowNamespace, cluster, namespace string) (*profile, error) {
	return wfc.profiles.find(workflowNamespace, cluster, namespace)
}

func (wfc *WorkflowController) clusterOf(obj metav1.Object) string {
	return common.Cluster(obj, wfc.primaryCluster())
}
