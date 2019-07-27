package template

import (
	"log"

	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Global variables
var (
	restConfig   *rest.Config
	clientConfig clientcmd.ClientConfig
	clientset    *kubernetes.Clientset
	wfClientset  *wfclientset.Clientset
	wftmplClient v1alpha1.WorkflowTemplateInterface
	namespace    string
)

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
		namespace, _, err = clientConfig.Namespace()
		if err != nil {
			log.Fatal(err)
		}
	}
	wfClientset = wfclientset.NewForConfigOrDie(restConfig)
	wftmplClient = wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace)
	return wftmplClient
}
