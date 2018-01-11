package e2e

import (
	"flag"

	"github.com/argoproj/argo/cmd/argo/commands"
	"github.com/argoproj/argo/workflow/common"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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

func newInstallArgs(namespace string) commands.InstallFlags {
	return commands.InstallFlags{
		ControllerName:  common.DefaultControllerDeploymentName,
		UIName:          commands.ArgoUIDeploymentName,
		Namespace:       namespace,
		ConfigMap:       common.DefaultConfigMapName(common.DefaultControllerDeploymentName),
		ControllerImage: "argoproj/workflow-controller:latest",
		UIImage:         "argoproj/argoui:latest",
		ExecutorImage:   "argoproj/argoexec:latest",
		ServiceAccount:  "",
	}
}

func createNamespaceForTest() string {
	clientset := getKubernetesClient()
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
	clientset := getKubernetesClient()
	deleteOptions := metav1.DeleteOptions{}
	return clientset.CoreV1().Namespaces().Delete(namespace, &deleteOptions)
}

func installArgoInNamespace(namespace string) {
	args := newInstallArgs(namespace)
	commands.Install(nil, args)
}
