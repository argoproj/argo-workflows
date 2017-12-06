package e2e

import (
	"flag"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeConfig = flag.String("kubeconfig", "", "Path to Kubernetes config file")

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getKubernetesClient() *kubernetes.Clientset {
	if *kubeConfig == "" {
		if home := homeDir(); home != "" {
			k := filepath.Join(home, ".kube", "config")
			kubeConfig = &k
		} else {
			panic("Failed to find kubeConfig")
		}
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
