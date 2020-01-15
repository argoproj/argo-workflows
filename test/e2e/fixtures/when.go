package fixtures

import (
	"fmt"
	"testing"
	"time"

	"github.com/argoproj/pkg/humanize"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type When struct {
	t                     *testing.T
	diagnostics           *Diagnostics
	wf                    *wfv1.Workflow
	wfTemplate            *wfv1.WorkflowTemplate
	cronWf                *wfv1.CronWorkflow
	client                v1alpha1.WorkflowInterface
	wfTemplateClient      v1alpha1.WorkflowTemplateInterface
	cronClient            v1alpha1.CronWorkflowInterface
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	workflowName          string
	wfTemplateName        string
	cronWorkflowName      string
}

func (w *When) SubmitWorkflow() *When {
	if w.wf == nil {
		w.t.Fatal("No workflow to submit")
	}
	log.WithField("test", w.t.Name()).Info("Submitting workflow")
	wf, err := w.client.Create(w.wf)
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.workflowName = wf.Name
	}
	log.WithField("test", w.t.Name()).Info("Workflow submitted")
	return w
}

func (w *When) CreateWorkflowTemplate() *When {
	if w.wfTemplate == nil {
		w.t.Fatal("No workflow template to create")
	}
	log.WithField("test", w.t.Name()).Info("Creating workflow template")
	wfTmpl, err := w.wfTemplateClient.Create(w.wfTemplate)
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.wfTemplateName = wfTmpl.Name
	}
	log.WithField("test", w.t.Name()).Info("Workflow template created")
	return w
}

func (w *When) CreateCronWorkflow() *When {
	if w.cronWf == nil {
		w.t.Fatal("No cron workflow to create")
	}
	log.WithField("test", w.t.Name()).Info("Creating cron workflow")
	cronWf, err := w.cronClient.Create(w.cronWf)
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.cronWorkflowName = cronWf.Name
	}
	log.WithField("test", w.t.Name()).Info("Cron workflow created")
	return w
}

func (w *When) WaitForWorkflow(timeout time.Duration) *When {
	logCtx := log.WithFields(log.Fields{"test": w.t.Name(), "workflow": w.workflowName})
	logCtx.Info("Waiting on workflow")
	opts := metav1.ListOptions{FieldSelector: fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", w.workflowName)).String()}
	watch, err := w.client.Watch(opts)
	if err != nil {
		w.t.Fatal(err)
	}
	defer watch.Stop()
	timeoutCh := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		timeoutCh <- true
	}()
	for {
		select {
		case event := <-watch.ResultChan():
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

func (w *When) Wait(timeout time.Duration) *When {
	logCtx := log.WithFields(log.Fields{"test": w.t.Name(), "cron workflow": w.cronWorkflowName})
	logCtx.Infof("Waiting for %s", humanize.Duration(timeout))
	time.Sleep(timeout)
	logCtx.Infof("Done waiting")
	return w
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
		t:                     w.t,
		diagnostics:           w.diagnostics,
		workflowName:          w.workflowName,
		wfTemplateName:        w.wfTemplateName,
		cronWorkflowName:      w.cronWorkflowName,
		client:                w.client,
		cronClient:            w.cronClient,
		offloadNodeStatusRepo: w.offloadNodeStatusRepo,
	}
}
