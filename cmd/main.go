package main

import (
	"context"
	"flag"
	"path/filepath"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	wfclientset := wfclientset.NewForConfigOrDie(config)

	update(wfclientset, "hello-world-4zfcw")
}

func update(wfclientset *wfclientset.Clientset, wfname string) {
	taskSet, _ := wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("argo").Get(context.Background(), wfname, v1.GetOptions{})
	taskSet.TypeMeta = v1.TypeMeta{
		Kind:       workflow.WorkflowTaskSetKind,
		APIVersion: workflow.APIVersion,
	}
	taskResult := v1alpha1.TaskResult{
		Phase:   v1alpha1.NodeSucceeded,
		Message: "http failed",
		Outputs: &v1alpha1.Outputs{
			Parameters: []v1alpha1.Parameter{
				{
					Name:  "message",
					Value: v1alpha1.AnyStringPtr("Welcome"),
				},
			},
		},
	}
	taskSet.Status = &v1alpha1.WorkflowTaskSetStatus{}
	taskSet.Status.Nodes = make(map[string]v1alpha1.TaskResult)
	for _, task := range taskSet.Spec.Templates {
		if _, ok := taskSet.Status.Nodes[task.NodeID]; !ok {
			taskSet.Status.Nodes[task.NodeID] = taskResult
			break
		}
	}
	wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("argo").Update(context.Background(), taskSet, v1.UpdateOptions{})
}
