package e2e

import (
	"flag"

	"github.com/argoproj/argo/cmd/argo/commands"
	"github.com/argoproj/argo/workflow/common"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeConfig = flag.String("kubeconfig", "", "Path to Kubernetes config file")

func getKubernetesClient() *kubernetes.Clientset {
	if *kubeConfig == "" {
		panic("Kubeconfig not provided")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientSet
}

func createNamespaceForTest() string {
	clientset := getKubernetesClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "argo-e2e-test-",
		},
	}
	cns, err := clientset.Core().Namespaces().Create(ns)
	if err != nil {
		panic(err)
	}

	return cns.Name
}

func deleteTestNamespace(namespace string) error {
	clientset := getKubernetesClient()
	deleteOptions := metav1.DeleteOptions{}
	return clientset.Core().Namespaces().Delete(namespace, &deleteOptions)
}

func installArgoInNamespace(namespace string) {
	args := commands.InstallFlags{
		ControllerName: common.DefaultControllerDeploymentName,
		UIName:         common.DefaultUiDeploymentName,
		Namespace:      namespace,
		ConfigMap:      common.DefaultConfigMapName(common.DefaultControllerDeploymentName),
		//TODO(shri): Use better defaults that don't need Makefiles
		ControllerImage: "argoproj/workflow-controller:v2.0.0-alpha3",
		UIImage:         "argoproj/argoui:v2.0.0-alpha3",
		ExecutorImage:   "argoproj/argoexec:v2.0.0-alpha3",
		ServiceAccount:  "",
	}

	commands.Install(nil, args)
}
