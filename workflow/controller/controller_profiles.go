package controller

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	authutil "github.com/argoproj/argo-workflows/v3/util/auth"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
)

func (wfc *WorkflowController) newDefaultProfiles(restConfig *rest.Config, kubernetesClient kubernetes.Interface, workflowClient wfclientset.Interface, metadataClient metadata.Interface) profiles {
	log.WithField("managedNamespace", wfc.GetManagedNamespace()).Info("Creating local profile")
	return profiles{
		localProfileKey: {
			policyDef:          wfc.localPolicyDef(),
			restConfig:         restConfig,
			kubernetesClient:   kubernetesClient,
			workflowClient:     workflowClient,
			metadataClient:     metadataClient,
			podInformer:        wfc.newPodInformer(kubernetesClient, common.LocalCluster, wfc.GetManagedNamespace()),
			podGCInformer:      wfc.newPodGCInformer(metadataClient, common.LocalCluster, wfc.GetManagedNamespace()),
			taskResultInformer: wfc.newWorkflowTaskResultInformer(workflowClient, common.LocalCluster),
		},
	}
}

func (wfc *WorkflowController) addProfile(secret *apiv1.Secret) error {
	policyKey := policyKeyForSecret(secret)
	if _, ok := wfc.profiles[policyKey]; ok {
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
	logs.AddK8SLogTransportWrapper(config)
	metrics.AddMetricsTransportWrapper(config)
	kubernetesClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	namespace := common.Namespace(secret)

	write, err := authutil.CanI(context.Background(), kubernetesClient, "create", "pods", namespace, "")
	if err != nil {
		return err
	}
	read, err := authutil.CanI(context.Background(), kubernetesClient, "list", "pods", namespace, "")
	if err != nil {
		return err
	}

	system := secret.Namespace == wfc.namespace
	// we start an informer for each read profile, but if the read profile is in the user namespace, this would allow
	// users to re-configure the controller, we disallow that
	read = read && system
	workflowNamespace, cluster := common.ClusterWorkflowNamespace(secret)
	misconfigured := (!write && !read) || (system && secret.Namespace != workflowNamespace)

	log.
		WithField("secretNamespace", secret.Namespace).
		WithField("workflowNamespace", workflowNamespace).
		WithField("cluster", cluster).
		WithField("namespace", namespace).
		WithField("read", read).
		WithField("write", write).
		WithField("misconfigured", misconfigured).
		Info("Profile configuration")

	if misconfigured {
		return fmt.Errorf("profile is misconfigured: either no read or write, or a user secret offering to be used in wrong namespace")
	}

	workflowClient, err := wfclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		return err
	}
	var act act
	if read {
		act = actRead
	}
	if write {
		act = act ^ actWrite
	}
	profile := &profile{
		policyDef: policyDef{
			workflowNamespace: workflowNamespace,
			cluster:           cluster,
			namespace:         namespace,
			act:               act,
		},
		done: func() {
		},
	}
	if write {
		profile.restConfig = config
		profile.kubernetesClient = kubernetesClient
		profile.workflowClient = workflowClient
		profile.metadataClient = metadataClient
	}
	if read {
		done := make(chan struct{})
		profile.podInformer = wfc.newPodInformer(kubernetesClient, cluster, namespace)
		profile.podGCInformer = wfc.newPodGCInformer(metadataClient, cluster, namespace)
		profile.taskResultInformer = wfc.newWorkflowTaskResultInformer(workflowClient, cluster)
		profile.done = func() { done <- struct{}{} }
		go profile.run(done)
	}

	wfc.profiles[policyKey] = profile

	return nil
}

func (wfc *WorkflowController) removeProfile(secret *apiv1.Secret) {
	policyKey := policyKeyForSecret(secret)
	profile, ok := wfc.profiles[policyKey]
	if !ok {
		return
	}
	profile.done()
	delete(wfc.profiles, policyKey)
}

func policyKeyForSecret(secret *apiv1.Secret) cache.ExplicitKey {
	return cache.ExplicitKey(fmt.Sprintf("%s,%s", secret.Namespace, secret.Name))
}

func (wfc *WorkflowController) newProfileInformer() cache.SharedIndexInformer {

	allowed, err := authutil.CanI(context.Background(), wfc.localProfile().kubernetesClient, "list", "secrets", wfc.GetManagedNamespace(), "")
	if err != nil {
		log.Fatal(err)
	}
	if !allowed {
		return nil
	}

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
				Error("failed to add profile from secret")
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

func (wfc *WorkflowController) localPolicyDef() policyDef {
	return policyDef{
		workflowNamespace: wfc.GetManagedNamespace(),
		cluster:           common.LocalCluster,
		namespace:         wfc.GetManagedNamespace(),
		act:               actRead ^ actWrite,
	}
}

func (wfc *WorkflowController) localProfile() *profile {
	return wfc.profiles.local()
}

func (wfc *WorkflowController) profile(workflowNamespace, cluster, namespace string, act act) (*profile, error) {
	return wfc.profiles.find(workflowNamespace, cluster, namespace, act)
}
