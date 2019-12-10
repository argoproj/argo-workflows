package cron

import (
	"log"

	versioned "github.com/argoproj/argo/pkg/client/clientset/versioned"
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
	wfClientset  *versioned.Clientset
	cronWfClient v1alpha1.CronWorkflowInterface
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

// InitCronWorkflowClient creates a new client for the Kubernetes WorkflowTemplate CRD.
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
	return cronWfClient
}
