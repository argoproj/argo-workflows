package main

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
)

func main() {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		panic(err)
	}
	config.QPS = 512
	namespace, _, _ := kubeConfig.Namespace()
	w := versioned.NewForConfigOrDie(config).ArgoprojV1alpha1().Workflows(namespace)

	wf := &wfv1.Workflow{}
	err = yaml.Unmarshal([]byte(`
metadata:
  generateName: stress-
  labels:
    stress: "true"
spec:
  arguments:
    parameters:
      - name: nodes
        value: "2"
      - name: sleep
        value: "30s"
  workflowTemplateRef:
    name: massive
`), wf)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		_, err := w.Create(ctx, wf, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
		print(i, " ")
	}
}
