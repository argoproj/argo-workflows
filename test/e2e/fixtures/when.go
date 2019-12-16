package fixtures

import (
	"fmt"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type When struct {
	t            *testing.T
	wf           *wfv1.Workflow
	client       v1alpha1.WorkflowInterface
	workflowName string
}

func (w *When) SubmitWorkflow() *When {
	log.WithField("test", w.t.Name()).Info("Submitting workflow")
	wf, err := w.client.Create(w.wf)
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.workflowName = wf.Name
	}
	return w
}

func (w *When) WaitForWorkflow(timeout time.Duration) *When {
	logCtx := log.WithFields(log.Fields{"test": w.t.Name(), "workflow": w.workflowName})
	logCtx.Info("Waiting on workflow")
	opts := metav1.ListOptions{FieldSelector: fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", w.workflowName)).String()}
	watchIf, err := w.client.Watch(opts)
	if err != nil {
		w.t.Fatal(err)
	}
	defer watchIf.Stop()
	timeoutCh := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		timeoutCh <- true
	}()
	for {
		select {
		case event := <-watchIf.ResultChan():
			wf, ok := event.Object.(*wfv1.Workflow)
			if ok {
				if !wf.Status.FinishedAt.IsZero() {
					return w
				}
			} else {
				logCtx.WithField("event", event).Warn("Did not get workflow event")
			}
		case <-timeoutCh:
			w.t.Fatalf("timeout after %v waiting for finish", timeout)
		}
	}
}

func (w *When) DeleteWorkflow() *When {
	log.WithField("test", w.t.Name()).WithField("workflow", w.workflowName).Info("Deleting")
	err := w.client.Delete(w.workflowName, nil)
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) Then() *Then {
	return &Then{
		t:            w.t,
		workflowName: w.workflowName,
		client:       w.client,
	}
}
