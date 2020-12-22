package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
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
	w := versioned.NewForConfigOrDie(config)
	workflows := w.ArgoprojV1alpha1().Workflows(namespace)
	err = workflows.DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "stress"})
	if err != nil {
		panic(err)
	}
	for i := 0; i < 10000; i++ {
		_, err := workflows.Create(&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "stress-",
				Labels: map[string]string{
					"stress": "true",
				},
			},
			Spec: wfv1.WorkflowSpec{
				Arguments: wfv1.Arguments{
					Parameters: []wfv1.Parameter{
						{Name: "sleep", Value: wfv1.AnyStringPtr("10s")},
					},
				},
				WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "massive-workflow"},
			},
		})
		if err != nil {
			panic(err)
		}
		print(i, ",")
	}

}
