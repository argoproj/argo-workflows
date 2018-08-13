package e2e

import (
	"flag"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var kubeConfig = flag.String("kubeconfig", "", "Path to Kubernetes config file")

func getKubernetesClient() (*rest.Config, *kubernetes.Clientset) {
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

	return config, clientSet
}

func createNamespaceForTest() string {
	_, clientset := getKubernetesClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "argo-e2e-test-",
		},
	}
	cns, err := clientset.CoreV1().Namespaces().Create(ns)
	if err != nil {
		panic(err)
	}
	return cns.Name
}

func deleteTestNamespace(namespace string) error {
	_, clientset := getKubernetesClient()
	deleteOptions := metav1.DeleteOptions{}
	return clientset.CoreV1().Namespaces().Delete(namespace, &deleteOptions)
}
