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

	cluster := common.PrimaryCluster()
	namespace := wfc.GetManagedNamespace()

	log.WithField("cluster", cluster).
		WithField("namespace", namespace).
		Info("Loading primary profile")

	wfc.profiles[cluster] = &profile{
		restConfig:         restConfig,
		kubernetesClient:   kubernetesClient,
		workflowClient:     workflowClient,
		metadataClient:     metadataClient,
		podInformer:        wfc.newPodInformer(kubernetesClient, cluster, namespace),
		podGCInformer:      wfc.newPodGCInformer(metadataClient, cluster, namespace),
		taskResultInformer: wfc.newWorkflowTaskResultInformer(workflowClient, cluster, namespace),
	}
}

func (wfc *WorkflowController) loadProfile(secret *apiv1.Secret) error {

	cluster := common.Cluster(secret)
	namespace := common.Namespace(secret)

	log.WithField("cluster", cluster).
		WithField("namespace", namespace).
		Info("Loading profile")

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

	workflowClient, err := wfclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		return err
	}
	p := &profile{
		restConfig:         config,
		kubernetesClient:   kubernetesClient,
		workflowClient:     workflowClient,
		metadataClient:     metadataClient,
		podInformer:        wfc.newPodInformer(kubernetesClient, cluster, namespace),
		podGCInformer:      wfc.newPodGCInformer(metadataClient, cluster, namespace),
		taskResultInformer: wfc.newWorkflowTaskResultInformer(workflowClient, cluster, namespace),
		done:               make(chan struct{}),
	}
	wfc.profiles[cluster] = p
	return nil
}

func (wfc *WorkflowController) primaryProfile() *profile {
	return wfc.profiles[common.PrimaryCluster()]
}

func (wfc *WorkflowController) profile(cluster string) (*profile, error) {
	return wfc.profiles.find(cluster)
}
