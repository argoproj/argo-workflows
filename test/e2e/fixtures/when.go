package fixtures

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type When struct {
	given *Given
}

func (w *When) SubmitWorkflow() *When {
	fmt.Printf("submitting %s\n", w.wf().Name)
	_, err := w.client().Create(w.wf())
	if err != nil {
		w.t().Fatal(err)
	}
	return w
}

func (w *When) WaitForWorkflow() *When {
	fmt.Printf("waiting for %s\n", w.wf().Name)
	wfClient := w.client()
	_, err := wfClient.Get(w.wf().Name, metav1.GetOptions{})
	opts := metav1.ListOptions{FieldSelector: fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", w.wf().Name)).String()}
	watchIf, err := wfClient.Watch(opts)
	if err != nil {
		w.t().Fatal(err)
	}
	defer watchIf.Stop()
	for {
		next := <-watchIf.ResultChan()
		wf, _ := next.Object.(*wfv1.Workflow)
		if !wf.Status.FinishedAt.IsZero() {
			return w
		}
	}
	return w
}

func (w *When) DeleteWorkflow() *When {
	fmt.Printf("deleting %s\n", w.wf().Name)
	err := w.client().Delete(w.given.wf.Name, nil)
	if err != nil {
		w.t().Fatal(err)
	}
	return w
}

func (w *When) Then() *Then {
	return &Then{w.given}
}

func (w *When) wf() *wfv1.Workflow {
	return w.given.wf
}

func (w *When) client() v1alpha1.WorkflowInterface {
	return w.given.client()
}

func (w *When) t() *testing.T {
	return w.given.t()
}
