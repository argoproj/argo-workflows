package fixtures

import (
	"testing"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type Then struct {
	t                     *testing.T
	diagnostics           *Diagnostics
	workflowName          string
	wfTemplateNames       []string
	cronWorkflowName      string
	client                v1alpha1.WorkflowInterface
	cronClient            v1alpha1.CronWorkflowInterface
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

func (t *Then) Expect(block func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus)) *Then {
	if t.workflowName == "" {
		t.t.Fatal("No workflow to test")
	}
	log.WithFields(log.Fields{"test": t.t.Name(), "workflow": t.workflowName}).Info("Checking expectation")
	wf, err := t.client.Get(t.workflowName, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	if wf.Status.IsOffloadNodeStatus() {
		offloadedNodes, err := t.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
		if err != nil {
			t.t.Fatal(err)
		}
		wf.Status.Nodes = offloadedNodes
	}
	block(t.t, &wf.ObjectMeta, &wf.Status)
	return t
}

func (t *Then) ExpectCron(block func(t *testing.T, cronWf *wfv1.CronWorkflow)) *Then {
	if t.cronWorkflowName == "" {
		t.t.Fatal("No cron workflow to test")
	}
	log.WithFields(log.Fields{"test": t.t.Name(), "cron workflow": t.cronWorkflowName}).Info("Checking expectation")
	cronWf, err := t.cronClient.Get(t.cronWorkflowName, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	block(t.t, cronWf)
	return t
}

func (t *Then) ExpectWorkflowList(listOptions metav1.ListOptions, block func(t *testing.T, wfList *wfv1.WorkflowList)) *Then {
	log.WithFields(log.Fields{"test": t.t.Name()}).Info("Getting relevant workflows")
	wfList, err := t.client.List(listOptions)
	if err != nil {
		t.t.Fatal(err)
	}
	log.WithFields(log.Fields{"test": t.t.Name()}).Info("Got relevant workflows")
	log.WithFields(log.Fields{"test": t.t.Name()}).Info("Checking expectation")
	block(t.t, wfList)
	return t
}

func (t *Then) RunCli(args []string, block func(t *testing.T, output string, err error)) *Then {
	output, err := runCli(t.diagnostics, args)
	block(t.t, output, err)
	return t
}
