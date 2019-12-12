package fixtures

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type When struct {
	t          *testing.T
	wf         *wfv1.Workflow
	client     v1alpha1.WorkflowInterface
	kubeClient kubernetes.Interface
	name       string
}

func (w *When) SubmitWorkflow() *When {
	fmt.Printf("submitting workflow\n")
	wf, err := w.client.Create(w.wf)
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.name = wf.Name
	}
	return w
}

func (w *When) WaitForWorkflow() *When {
	fmt.Printf("waiting for %s\n", w.name)
	watchIf, err := w.client.Watch(metav1.ListOptions{FieldSelector: fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", w.name)).String()})
	if err != nil {
		w.t.Fatal(err)
	}
	defer watchIf.Stop()
	for {
		next := <-watchIf.ResultChan()
		wf, _ := next.Object.(*wfv1.Workflow)
		if !wf.Status.FinishedAt.IsZero() {
			return w
		}
	}
}

func (w *When) DeleteWorkflow() *When {
	fmt.Printf("deleting %s\n", w.name)
	err := w.client.Delete(w.name, nil)
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) Then() *Then {
	return &Then{
		t:          w.t,
		name:       w.name,
		client:     w.client,
		kubeClient: w.kubeClient,
	}
}
