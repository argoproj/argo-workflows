package fixtures

import (
	"testing"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type Then struct {
	t            *testing.T
	workflowName string
	client       v1alpha1.WorkflowInterface
}

func (t *Then) Expect(block func(t *testing.T, wf *wfv1.WorkflowStatus)) *Then {
	log.WithFields(log.Fields{"test": t.t.Name(), "workflow": t.workflowName}).Info("Checking expectation")
	wf, err := t.client.Get(t.workflowName, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	block(t.t, &wf.Status)
	return t
}
