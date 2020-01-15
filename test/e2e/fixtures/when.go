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
	"github.com/argoproj/argo/workflow/packer"
)

type When struct {
	t                     *testing.T
	diagnostics           *Diagnostics
	wf                    *wfv1.Workflow
	wfTemplates           []*wfv1.WorkflowTemplate
	cronWf                *wfv1.CronWorkflow
	client                v1alpha1.WorkflowInterface
	wfTemplateClient      v1alpha1.WorkflowTemplateInterface
	cronClient            v1alpha1.CronWorkflowInterface
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	workflowName          string
	wfTemplateNames       []string
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

func (w *When) CreateWorkflowTemplates() *When {
	if len(w.wfTemplates) == 0 {
		w.t.Fatal("No workflow templates to create")
	}
	for _, wfTmpl := range w.wfTemplates {
		log.WithField("test", w.t.Name()).Infof("Creating workflow template %s", wfTmpl.Name)
		wfTmpl, err := w.wfTemplateClient.Create(wfTmpl)
		if err != nil {
			w.t.Fatal(err)
		} else {
			w.wfTemplateNames = append(w.wfTemplateNames, wfTmpl.Name)
		}
		log.WithField("test", w.t.Name()).Infof("Workflow template created %s", wfTmpl.Name)
	}
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

func (w *When) WaitForWorkflowCondition(condition func(wf *wfv1.Workflow) bool, timeout time.Duration) *When {
	logCtx := log.WithFields(log.Fields{"test": w.t.Name(), "workflow": w.workflowName})
	logCtx.Info("Waiting for workflow condition")
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
			logCtx.WithField("type", event.Type).Info("Event")
			wf, ok := event.Object.(*wfv1.Workflow)
			if ok {
				w.hydrateWorkflow(wf)
				if condition(wf) {
					return w
				}
			}
		case <-timeoutCh:
			w.t.Fatalf("timeout after %v waiting for condition", timeout)
		}
	}
}

func (w *When) hydrateWorkflow(wf *wfv1.Workflow) {
	err := packer.DecompressWorkflow(wf)
	if err != nil {
		w.t.Fatal(err)
	}
	if wf.Status.IsOffloadNodeStatus() {
		offloadedNodes, err := w.offloadNodeStatusRepo.Get(wf.Name, wf.Namespace, wf.GetOffloadNodeStatusVersion())
		if err != nil {
			w.t.Fatal(err)
		}
		wf.Status.Nodes = offloadedNodes
	}
}
func (w *When) WaitForWorkflowStarted(timeout time.Duration) *When {
	return w.WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
		return !wf.Status.StartedAt.IsZero()
	}, timeout)
}

func (w *When) WaitForWorkflow(timeout time.Duration) *When {
	return w.WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
		return !wf.Status.FinishedAt.IsZero()
	}, timeout)
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
		wfTemplateNames:       w.wfTemplateNames,
		cronWorkflowName:      w.cronWorkflowName,
		client:                w.client,
		cronClient:            w.cronClient,
		offloadNodeStatusRepo: w.offloadNodeStatusRepo,
	}
}
