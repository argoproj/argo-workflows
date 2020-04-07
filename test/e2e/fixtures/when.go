package fixtures

import (
	"fmt"
	"testing"
	"time"

	"github.com/argoproj/pkg/humanize"
	"k8s.io/client-go/kubernetes"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/test/util"
	"github.com/argoproj/argo/workflow/packer"
)

type When struct {
	t                     *testing.T
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
	kubeClient            kubernetes.Interface
	resourceQuota         *corev1.ResourceQuota
}

func (w *When) SubmitWorkflow() *When {
	if w.wf == nil {
		w.t.Fatal("No workflow to submit")
	}
	log.WithFields(log.Fields{"workflow": w.wf.Name}).Info("Submitting workflow")
	wf, err := w.client.Create(w.wf)
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.workflowName = wf.Name
	}
	log.WithFields(log.Fields{"workflow": wf.Name, "uid": wf.UID}).Info("Workflow submitted")
	return w
}

func (w *When) CreateWorkflowTemplates() *When {
	if len(w.wfTemplates) == 0 {
		w.t.Fatal("No workflow templates to create")
	}
	for _, wfTmpl := range w.wfTemplates {
		log.WithField("template", wfTmpl.Name).Info("Creating workflow template")
		wfTmpl, err := w.wfTemplateClient.Create(wfTmpl)
		if err != nil {
			w.t.Fatal(err)
		} else {
			w.wfTemplateNames = append(w.wfTemplateNames, wfTmpl.Name)
		}
		log.WithField("template", wfTmpl.Name).Info("Workflow template created")
	}
	return w
}

func (w *When) CreateCronWorkflow() *When {
	if w.cronWf == nil {
		w.t.Fatal("No cron workflow to create")
	}
	log.WithField("cronWorkflow", w.cronWf.Name).Info("Creating cron workflow")
	cronWf, err := w.cronClient.Create(w.cronWf)
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.cronWorkflowName = cronWf.Name
	}
	log.WithField("uid", cronWf.UID).Info("Cron workflow created")
	return w
}

func (w *When) WaitForWorkflowCondition(test func(wf *wfv1.Workflow) bool, condition string, duration time.Duration) *When {
	return w.waitForWorkflow(w.workflowName, test, condition, duration)
}

func (w *When) waitForWorkflow(workflowName string, test func(wf *wfv1.Workflow) bool, condition string, timeout time.Duration) *When {
	logCtx := log.WithFields(log.Fields{"workflow": workflowName, "condition": condition, "timeout": timeout})
	logCtx.Info("Waiting for condition")
	opts := metav1.ListOptions{FieldSelector: fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", workflowName)).String()}
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
				logCtx.WithFields(log.Fields{"type": event.Type, "phase": wf.Status.Phase, "message": wf.Status.Message}).Info("...")
				w.hydrateWorkflow(wf)
				if test(wf) {
					logCtx.Infof("Condition met")
					return w
				}
			} else {
				w.t.Fatal("not ok")
			}
		case <-timeoutCh:
			w.t.Fatalf("timeout after %v waiting for condition %s", timeout, condition)
		}
	}
}

func (w *When) hydrateWorkflow(wf *wfv1.Workflow) {
	err := packer.DecompressWorkflow(wf)
	if err != nil {
		w.t.Fatal(err)
	}
	if wf.Status.IsOffloadNodeStatus() && w.offloadNodeStatusRepo.IsEnabled() {
		offloadedNodes, err := w.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
		if err != nil {
			w.t.Fatal(err)
		}
		wf.Status.Nodes = offloadedNodes
	}
}
func (w *When) WaitForWorkflowToStart(timeout time.Duration) *When {
	return w.waitForWorkflow(w.workflowName, func(wf *wfv1.Workflow) bool {
		return !wf.Status.StartedAt.IsZero()
	}, "to start", timeout)
}

func (w *When) WaitForWorkflow(timeout time.Duration) *When {
	return w.waitForWorkflow(w.workflowName, func(wf *wfv1.Workflow) bool {
		return !wf.Status.FinishedAt.IsZero()
	}, "to finish", timeout)
}

func (w *When) WaitForWorkflowName(workflowName string, timeout time.Duration) *When {
	return w.waitForWorkflow(workflowName, func(wf *wfv1.Workflow) bool {
		return !wf.Status.FinishedAt.IsZero()
	}, "to finish", timeout)
}

func (w *When) Wait(timeout time.Duration) *When {
	logCtx := log.WithFields(log.Fields{"cronWorkflow": w.cronWorkflowName})
	logCtx.Infof("Waiting for %s", humanize.Duration(timeout))
	time.Sleep(timeout)
	logCtx.Infof("Done waiting")
	return w
}

func (w *When) DeleteWorkflow() *When {
	log.WithField("workflow", w.workflowName).Info("Deleting")
	err := w.client.Delete(w.workflowName, nil)
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) RunCli(args []string, block func(t *testing.T, output string, err error)) *When {
	output, err := runCli("../../dist/argo", append([]string{"-n", Namespace}, args...)...)
	block(w.t, output, err)
	if w.t.Failed() {
		w.t.FailNow()
	}
	return w
}

func (w *When) MemoryQuota(quota string) *When {
	obj, err := util.CreateHardMemoryQuota(w.kubeClient, "argo", "memory-quota", quota)
	if err != nil {
		w.t.Fatal(err)
	}
	w.resourceQuota = obj
	return w
}

func (w *When) DeleteQuota() *When {
	err := util.DeleteQuota(w.kubeClient, w.resourceQuota)
	if err != nil {
		w.t.Fatal(err)
	}
	w.resourceQuota = nil
	return w
}

func (w *When) Then() *Then {
	return &Then{
		t:                     w.t,
		workflowName:          w.workflowName,
		wfTemplateNames:       w.wfTemplateNames,
		cronWorkflowName:      w.cronWorkflowName,
		client:                w.client,
		cronClient:            w.cronClient,
		offloadNodeStatusRepo: w.offloadNodeStatusRepo,
		kubeClient:            w.kubeClient,
	}
}

func (w *When) Given() *Given {
	return &Given{
		t:                     w.t,
		client:                w.client,
		wfTemplateClient:      w.wfTemplateClient,
		cronClient:            w.cronClient,
		offloadNodeStatusRepo: w.offloadNodeStatusRepo,
		wf:                    w.wf,
		wfTemplates:           w.wfTemplates,
		cronWf:                w.cronWf,
		workflowName:          w.workflowName,
		kubeClient:            w.kubeClient,
	}
}
