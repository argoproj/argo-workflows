package template

import (
	"github.com/argoproj/argo/cmd/argo/commands/client"
	"log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	versioned "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/templateresolution"
)

// Global variables
var (
	restConfig   *rest.Config
	clientConfig clientcmd.ClientConfig
	clientset    *kubernetes.Clientset
	wfClientset  *versioned.Clientset
	wftmplClient v1alpha1.WorkflowTemplateInterface
	namespace    string
)

func initKubeClient() *kubernetes.Clientset {
	if clientset != nil {
		return clientset
	}
	var err error
	restConfig, err = client.Config.ClientConfig()
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

// InitWorkflowTemplateClient creates a new client for the Kubernetes WorkflowTemplate CRD.
func InitWorkflowTemplateClient(ns ...string) v1alpha1.WorkflowTemplateInterface {
	if wftmplClient != nil {
		return wftmplClient
	}
	initKubeClient()
	var err error
	if len(ns) > 0 {
		namespace = ns[0]
	} else {
		namespace, _, err = client.Config.Namespace()
		if err != nil {
			log.Fatal(err)
		}
	}
	wfClientset = versioned.NewForConfigOrDie(restConfig)
	wftmplClient = wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace)
	return wftmplClient
}

// LazyWorkflowTemplateGetter is a wrapper of v1alpha1.WorkflowTemplateInterface which
// supports lazy initialization.
type LazyWorkflowTemplateGetter struct{}

// Get initializes it just before it's actually used and returns a retrieved workflow template.
func (c LazyWorkflowTemplateGetter) Get(name string) (*wfv1.WorkflowTemplate, error) {
	if wftmplClient == nil {
		_ = InitWorkflowTemplateClient()
	}
	return templateresolution.WrapWorkflowTemplateInterface(wftmplClient).Get(name)
}

var _ templateresolution.WorkflowTemplateNamespacedGetter = &LazyWorkflowTemplateGetter{}
