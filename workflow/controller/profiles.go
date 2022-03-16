package controller

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
)

func (wfc *WorkflowController) addProfile(secret *apiv1.Secret) error {
	cluster, namespace, profileName := clusterProfile(secret)
	if _, ok := wfc.profiles[profileName]; ok {
		return nil
	}
	kc, err := clientcmd.Load(secret.Data["kubeconfig"])
	if err != nil {
		return err
	}
	config, err := clientcmd.NewNonInteractiveClientConfig(*kc, kc.CurrentContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()).ClientConfig()
	if err != nil {
		return err
	}
	log.WithField("cluster", cluster).Info("creating clients")
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

	done := make(chan struct{})

	wfc.profiles[profileName] = &profile{
		cluster:            cluster,
		namespace:          namespace,
		restConfig:         config,
		kubernetesClient:   kubernetesClient,
		workflowClient:     workflowClient,
		metadataClient:     metadataClient,
		podInformer:        wfc.newPodInformer(kubernetesClient, cluster, namespace),
		podGCInformer:      wfc.newPodGCInformer(metadataClient, cluster, namespace),
		taskResultInformer: wfc.newWorkflowTaskResultInformer(workflowClient, cluster),
		done:               done,
	}

	wfc.profiles[profileName].podInformer.Run(done)

	return nil
}

func clusterProfile(secret *apiv1.Secret) (string, string, string) {
	workflowNamespace := common.MetaWorkflowNamespace(secret)
	cluster := secret.Labels[common.LabelKeyCluster]
	namespace := secret.Labels[common.LabelKeyNamespace]
	return cluster, namespace, profileName(workflowNamespace, cluster, namespace)
}

func (wfc *WorkflowController) removeProfile(secret *apiv1.Secret) {
	_, _, profileName := clusterProfile(secret)
	if _, ok := wfc.profiles[profileName]; ok {
		return
	}
	wfc.profiles[profileName].done <- struct{}{}

	delete(wfc.profiles, profileName)

}

func (wfc *WorkflowController) newProfileInformer() cache.SharedIndexInformer {
	informer := v1.NewFilteredSecretInformer(
		wfc.localProfile().kubernetesClient,
		wfc.GetManagedNamespace(),
		20*time.Minute,
		cache.Indexers{},
		func(options *metav1.ListOptions) {
			options.LabelSelector = common.LabelKeyCluster
		},
	)

	addFunc := func(obj interface{}) {
		secret := obj.(*apiv1.Secret)
		if err := wfc.addProfile(secret); err != nil {
			log.WithError(err).
				WithField("namespace", secret.Namespace).
				WithField("name", secret.Name).
				Error("failed to load cluster secret")
		}
	}
	removeFunc := func(obj interface{}) {
		secret := obj.(*apiv1.Secret)
		wfc.removeProfile(secret)
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: addFunc,
		UpdateFunc: func(_, obj interface{}) {
			addFunc(obj)
		},
		DeleteFunc: removeFunc,
	})

	return informer
}

func profileName(workflowNamespace, cluster, namespace string) string {
	return workflowNamespace + "," + cluster + "," + namespace
}

func (wfc *WorkflowController) localProfileName() string {
	return ""
}

func (wfc *WorkflowController) localProfile() *profile {
	return wfc.profiles[wfc.localProfileName()]
}

func (wfc *WorkflowController) profile(workflowNamespace, cluster, namespace string) (*profile, error) {
	for _, p := range wfc.profiles {
		if cluster == p.cluster &&
			(workflowNamespace == p.workflowNamespace || p.workflowNamespace == "") &&
			(namespace == p.namespace || p.namespace == "") {
			log.Infof("%s,%s,%s -> %s,%s,%s", workflowNamespace, cluster, namespace, p.workflowNamespace, p.cluster, p.namespace)
			return p, nil
		}
	}
	return nil, fmt.Errorf("profile not found for %s,%s,%s", workflowNamespace, cluster, namespace)
}

func (woc *wfOperationCtx) profile(cluster, namespace string) (*profile, error) {
	return woc.controller.profile(woc.wf.Namespace, cluster, namespace)
}
