package main

import (
	"context"
	"flag"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func main() {
	ctx := logging.TestContext(context.Background())
	logger := logging.RequireLoggerFromContext(ctx)

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
	err = w.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "stress"})
	if err != nil {
		panic(err)
	}

	wf := &wfv1.Workflow{}
	err = yaml.Unmarshal([]byte(`
metadata:
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

	n := 0
	flag.IntVar(&n, "n", 1, "number of workflows")
	flag.Parse()

	logger.WithField("workflowCount", n).Info(ctx, "running workflows")

	for i := 0; i < n; i++ {
		wf.SetName(fmt.Sprintf("stress-%d", i))
		_, err := w.Create(ctx, wf, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
		logger.WithField("workflowName", wf.GetName()).Info(ctx, "created workflow")
	}
}
