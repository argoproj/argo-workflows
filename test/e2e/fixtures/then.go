package fixtures

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/hydrator"
)

type Then struct {
	t                *testing.T
	workflowName     string
	wfTemplateNames  []string
	cronWorkflowName string
	client           v1alpha1.WorkflowInterface
	cronClient       v1alpha1.CronWorkflowInterface
	hydrator         hydrator.Interface
	kubeClient       kubernetes.Interface
}

func (t *Then) ExpectWorkflow(block func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus)) *Then {
	return t.expectWorkflow(t.workflowName, block)
}

func (t *Then) ExpectWorkflowName(workflowName string, block func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus)) *Then {
	return t.expectWorkflow(workflowName, block)
}

func (t *Then) expectWorkflow(workflowName string, block func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus)) *Then {
	if workflowName == "" {
		t.t.Fatal("No workflow to test")
	}
	log.WithFields(log.Fields{"test": t.t.Name(), "workflow": workflowName}).Info("Checking expectation")
	wf, err := t.client.Get(workflowName, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	err = t.hydrator.Hydrate(wf)
	if err != nil {
		t.t.Fatal(err)
	}
	block(t.t, &wf.ObjectMeta, &wf.Status)
	if t.t.Failed() {
		t.t.FailNow()
	}
	return t

}

func (t *Then) ExpectCron(block func(t *testing.T, cronWf *wfv1.CronWorkflow)) *Then {
	if t.cronWorkflowName == "" {
		t.t.Fatal("No cron workflow to test")
	}
	log.WithFields(log.Fields{"cronWorkflow": t.cronWorkflowName}).Info("Checking cron expectation")
	cronWf, err := t.cronClient.Get(t.cronWorkflowName, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	block(t.t, cronWf)
	if t.t.Failed() {
		t.t.FailNow()
	}
	return t
}

func (t *Then) ExpectWorkflowList(listOptions metav1.ListOptions, block func(t *testing.T, wfList *wfv1.WorkflowList)) *Then {
	log.Info("Listing workflows")
	wfList, err := t.client.List(listOptions)
	if err != nil {
		t.t.Fatal(err)
	}
	log.Info("Checking expectation")
	block(t.t, wfList)
	if t.t.Failed() {
		t.t.FailNow()
	}
	return t
}

func (t *Then) expectAuditEvents(block func(*testing.T, []apiv1.Event)) *Then {
	eventList, err := t.kubeClient.CoreV1().Events(Namespace).List(metav1.ListOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	var events []apiv1.Event
	for _, e := range eventList.Items {
		if e.Namespace == Namespace && e.InvolvedObject.Kind == workflow.WorkflowKind {
			events = append(events, e)
		}
	}
	log.WithFields(log.Fields{"event": events}).Debug("Events")
	block(t.t, events)
	if t.t.Failed() {
		t.t.FailNow()
	}
	return t
}

func (t *Then) ExpectAuditEvent(f func(apiv1.Event) bool) *Then {
	return t.expectAuditEvents(func(t *testing.T, events []apiv1.Event) {
		for _, item := range events {
			if f(item) {
				return
			}
		}
		assert.Fail(t, "did not see expected event")
	})
}

func (t *Then) RunCli(args []string, block func(t *testing.T, output string, err error)) *Then {
	output, err := runCli("../../dist/argo", append([]string{"-n", Namespace}, args...)...)
	block(t.t, output, err)
	if t.t.Failed() {
		t.t.FailNow()
	}
	return t
}

func (t *Then) When() *When {
	return &When{
		t:                t.t,
		client:           t.client,
		cronClient:       t.cronClient,
		hydrator:         t.hydrator,
		workflowName:     t.workflowName,
		wfTemplateNames:  t.wfTemplateNames,
		cronWorkflowName: t.cronWorkflowName,
		kubeClient:       t.kubeClient,
	}
}
