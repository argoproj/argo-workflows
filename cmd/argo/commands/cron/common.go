package cron

import (
	"log"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/templateresolution"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

// Global variables
var (
	// DEPRECATED
	restConfig *rest.Config
	// DEPRECATED
	clientConfig clientcmd.ClientConfig
	// DEPRECATED
	clientset *kubernetes.Clientset
	// DEPRECATED
	wfClientset *versioned.Clientset
	// DEPRECATED
	cronWfClient v1alpha1.CronWorkflowInterface
	// DEPRECATED
	wftmplClient v1alpha1.WorkflowTemplateInterface
	// DEPRECATED
	namespace string
)

// DEPRECATED
func initKubeClient() *kubernetes.Clientset {
	if clientset != nil {
		return clientset
	}
	var err error
	restConfig, err = clientConfig.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	return clientset
}

// InitCronWorkflowClient creates a new client for the Kubernetes WorkflowTemplate CRD.
// DEPRECATED
func InitCronWorkflowClient(ns ...string) v1alpha1.CronWorkflowInterface {
	if cronWfClient != nil {
		return cronWfClient
	}
	initKubeClient()
	var err error
	if len(ns) > 0 {
		namespace = ns[0]
	} else {
		namespace, _, err = clientConfig.Namespace()
		if err != nil {
			log.Fatal(err)
		}
	}
	wfClientset = versioned.NewForConfigOrDie(restConfig)
	cronWfClient = wfClientset.ArgoprojV1alpha1().CronWorkflows(namespace)
	wftmplClient = wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace)
	return cronWfClient
}

// LazyWorkflowTemplateGetter is a wrapper of v1alpha1.WorkflowTemplateInterface which
// supports lazy initialization.
// DEPRECATED
type LazyWorkflowTemplateGetter struct{}

// Get initializes it just before it's actually used and returns a retrieved workflow template.
// DEPRECATED
func (c LazyWorkflowTemplateGetter) Get(name string) (*wfv1.WorkflowTemplate, error) {
	if wftmplClient == nil {
		_ = InitCronWorkflowClient()
	}
	return templateresolution.WrapWorkflowTemplateInterface(wftmplClient).Get(name)
}

// DEPRECATED
var _ templateresolution.WorkflowTemplateNamespacedGetter = &LazyWorkflowTemplateGetter{}
