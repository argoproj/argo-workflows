package fixtures

import (
	"os"
	"os/exec"
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	argoexec "github.com/argoproj/pkg/exec"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Then struct {
	t                *testing.T
	workflowName     string
	cronWorkflowName string
	client           v1alpha1.WorkflowInterface
	cronClient       v1alpha1.CronWorkflowInterface
}

func (t *Then) Expect(block func(*testing.T, *wfv1.WorkflowStatus)) *Then {
	if t.workflowName == "" {
		t.t.Fatal("No workflow to test")
	}
	log.WithFields(log.Fields{"test": t.t.Name(), "workflow": t.workflowName}).Info("Checking expectation")
	wf, err := t.client.Get(t.workflowName, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	block(t.t, &wf.Status)
	return t
}

func (t *Then) ExpectCron(block func(*testing.T, *wfv1.CronWorkflow)) *Then {
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

func (t *Then) ExpectWorkflowList(listOptions metav1.ListOptions, block func(*testing.T, *wfv1.WorkflowList)) *Then {
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

func (t *Then) RunCli(args []string, block func(*testing.T, string)) *Then {
	cmd := exec.Command("../../../dist/argo", args...)
	cmd.Env = os.Environ()
	cmd.Dir = ""

	output, err := argoexec.RunCommandExt(cmd, argoexec.CmdOpts{})
	if err != nil {
		t.t.Fatal(err)
	}

	block(t.t, output)
	return t
}
