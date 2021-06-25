package fixtures

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
)

type When struct {
	t                 *testing.T
	wf                *wfv1.Workflow
	wfeb              *wfv1.WorkflowEventBinding
	wfTemplates       []*wfv1.WorkflowTemplate
	cwfTemplates      []*wfv1.ClusterWorkflowTemplate
	cronWf            *wfv1.CronWorkflow
	client            v1alpha1.WorkflowInterface
	wfebClient        v1alpha1.WorkflowEventBindingInterface
	wfTemplateClient  v1alpha1.WorkflowTemplateInterface
	cwfTemplateClient v1alpha1.ClusterWorkflowTemplateInterface
	cronClient        v1alpha1.CronWorkflowInterface
	hydrator          hydrator.Interface
	kubeClient        kubernetes.Interface
	bearerToken       string
}

func (w *When) SubmitWorkflow() *When {
	w.t.Helper()
	if w.wf == nil {
		w.t.Fatal("No workflow to submit")
	}
	_, _ = fmt.Println("Submitting workflow", w.wf.Name, w.wf.GenerateName)
	ctx := context.Background()
	wf, err := w.client.Create(ctx, w.wf, metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.wf = wf
	}
	return w
}

func (w *When) SubmitWorkflowsFromWorkflowTemplates() *When {
	w.t.Helper()
	ctx := context.Background()
	for _, tmpl := range w.wfTemplates {
		_, _ = fmt.Println("Submitting workflow from workflow template", tmpl.Name)
		wf, err := w.client.Create(ctx, common.NewWorkflowFromWorkflowTemplate(tmpl.Name, tmpl.Spec.WorkflowMetadata, false), metav1.CreateOptions{})
		if err != nil {
			w.t.Fatal(err)
		} else {
			w.wf = wf
		}
	}
	return w
}

func (w *When) SubmitWorkflowsFromClusterWorkflowTemplates() *When {
	w.t.Helper()
	ctx := context.Background()
	for _, tmpl := range w.cwfTemplates {
		_, _ = fmt.Println("Submitting workflow from cluster workflow template", tmpl.Name)
		wf, err := w.client.Create(ctx, common.NewWorkflowFromWorkflowTemplate(tmpl.Name, tmpl.Spec.WorkflowMetadata, true), metav1.CreateOptions{})
		if err != nil {
			w.t.Fatal(err)
		} else {
			w.wf = wf
		}
	}
	return w
}

func (w *When) SubmitWorkflowsFromCronWorkflows() *When {
	w.t.Helper()
	_, _ = fmt.Println("Submitting workflow from cron workflow", w.cronWf.Name)
	ctx := context.Background()
	wf, err := w.client.Create(ctx, common.ConvertCronWorkflowToWorkflow(w.cronWf), metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.wf = wf
	}
	return w
}

func (w *When) CreateWorkflowEventBinding() *When {
	w.t.Helper()
	if w.wfeb == nil {
		w.t.Fatal("No workflow event to create")
	}
	_, _ = fmt.Println("Creating workflow event binding")
	ctx := context.Background()
	_, err := w.wfebClient.Create(ctx, w.wfeb, metav1.CreateOptions{})
	if err != nil {
		w.t.Error(err)
	}
	time.Sleep(1 * time.Second)
	return w
}

func (w *When) CreateWorkflowTemplates() *When {
	w.t.Helper()
	if len(w.wfTemplates) == 0 {
		w.t.Fatal("No workflow templates to create")
	}

	ctx := context.Background()
	for _, wfTmpl := range w.wfTemplates {
		_, _ = fmt.Println("Creating workflow template", wfTmpl.Name)
		_, err := w.wfTemplateClient.Create(ctx, wfTmpl, metav1.CreateOptions{})
		if err != nil {
			w.t.Fatal(err)
		}
	}
	time.Sleep(1 * time.Second)
	return w
}

func (w *When) CreateClusterWorkflowTemplates() *When {
	w.t.Helper()
	if len(w.cwfTemplates) == 0 {
		w.t.Fatal("No cluster workflow templates to create")
	}

	ctx := context.Background()
	for _, cwfTmpl := range w.cwfTemplates {
		_, _ = fmt.Println("Creating cluster workflow template", cwfTmpl.Name)
		_, err := w.cwfTemplateClient.Create(ctx, cwfTmpl, metav1.CreateOptions{})
		if err != nil {
			w.t.Fatal(err)
		}
	}
	time.Sleep(1 * time.Second)
	return w
}

func (w *When) CreateCronWorkflow() *When {
	w.t.Helper()
	if w.cronWf == nil {
		w.t.Fatal("No cron workflow to create")
	}
	_, _ = fmt.Println("Creating cron workflow", w.cronWf.Name)

	ctx := context.Background()
	cronWf, err := w.cronClient.Create(ctx, w.cronWf, metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.cronWf = cronWf
	}
	time.Sleep(1 * time.Second)
	return w
}

type Condition func(wf *wfv1.Workflow) (bool, string)

var (
	ToBeRunning             = ToHavePhase(wfv1.WorkflowRunning)
	ToBeSucceeded           = ToHavePhase(wfv1.WorkflowSucceeded)
	ToBeErrored             = ToHavePhase(wfv1.WorkflowError)
	ToBeFailed              = ToHavePhase(wfv1.WorkflowFailed)
	ToBeCompleted Condition = func(wf *wfv1.Workflow) (bool, string) {
		return wf.Labels[common.LabelKeyCompleted] == "true", "to be completed"
	}
	ToStart          Condition = func(wf *wfv1.Workflow) (bool, string) { return !wf.Status.StartedAt.IsZero(), "to start" }
	ToHaveRunningPod Condition = func(wf *wfv1.Workflow) (bool, string) {
		return wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
			return node.Type == wfv1.NodeTypePod && node.Phase == wfv1.NodeRunning
		}), "to have running pod"
	}
)

// `ToBeDone` replaces `ToFinish` which also makes sure the workflow is both complete not pending archiving.
// This additional check is not needed for most use case, however in `AfterTest` we delete the workflow and this
// creates a lot of warning messages in the logs that are cause by misuse rather than actual problems.
var ToBeDone Condition = func(wf *wfv1.Workflow) (bool, string) {
	toBeCompleted, _ := ToBeCompleted(wf)
	return toBeCompleted && wf.Labels[common.LabelKeyWorkflowArchivingStatus] != "Pending", "to be done"
}

var ToBeArchived Condition = func(wf *wfv1.Workflow) (bool, string) {
	return wf.Labels[common.LabelKeyWorkflowArchivingStatus] == "Archived", "to be archived"
}

var ToHavePhase = func(p wfv1.WorkflowPhase) Condition {
	return func(wf *wfv1.Workflow) (bool, string) {
		return wf.Status.Phase == p && wf.Labels[common.LabelKeyWorkflowArchivingStatus] != "Pending", fmt.Sprintf("to be %s", p)
	}
}

var ToBeWaitingOnAMutex Condition = func(wf *wfv1.Workflow) (bool, string) {
	return wf.Status.Synchronization != nil && wf.Status.Synchronization.Mutex != nil, "to be waiting on a mutex"
}

// Wait for a workflow to meet a condition:
// Options:
// * `time.Duration` - change the timeout - 30s by default
// * `string` - either:
//    * the workflow's name (not spaces)
//    * or a new message (if it contain spaces) - default "to finish"
// * `Condition` - a condition - `ToFinish` by default
func (w *When) WaitForWorkflow(options ...interface{}) *When {
	w.t.Helper()
	timeout := defaultTimeout
	workflowName := ""
	if w.wf != nil {
		workflowName = w.wf.Name
	}
	condition := ToBeDone
	for _, opt := range options {
		switch v := opt.(type) {
		case time.Duration:
			timeout = v - 30*time.Second + defaultTimeout
		case string:
			workflowName = v
		case Condition:
			condition = v
		default:
			w.t.Fatal("unknown option type: " + reflect.TypeOf(opt).String())
		}
	}

	start := time.Now()

	fieldSelector := ""
	if workflowName != "" {
		fieldSelector = "metadata.name=" + workflowName
	}

	_, _ = fmt.Println("Waiting", timeout.String(), "for workflow", fieldSelector)

	ctx := context.Background()
	opts := metav1.ListOptions{LabelSelector: Label, FieldSelector: fieldSelector}
	watch, err := w.client.Watch(ctx, opts)
	if err != nil {
		w.t.Error(err)
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
				w.hydrateWorkflow(wf)
				printWorkflow(wf)
				if ok, message := condition(wf); ok {
					_, _ = fmt.Printf("Condition %q met after %s\n", message, time.Since(start).Truncate(time.Second))
					w.wf = wf
					return w
				}
				// once done the workflow is done, the condition can never be met
				// rather than wait maybe 30s for something that can never happen
				if ok, _ = ToBeDone(wf); ok {
					w.t.Errorf("condition never and cannot be met because the workflow is done")
					return w
				}
			} else {
				w.t.Errorf("not ok")
				return w
			}
		case <-timeoutCh:
			w.t.Errorf("timeout after %v waiting for condition", timeout)
			return w
		}
	}
}

func (w *When) hydrateWorkflow(wf *wfv1.Workflow) {
	w.t.Helper()
	err := w.hydrator.Hydrate(wf)
	if err != nil {
		w.t.Fatal(err)
	}
}

func (w *When) Wait(timeout time.Duration) *When {
	w.t.Helper()
	_, _ = fmt.Println("Waiting for", timeout.String())
	time.Sleep(timeout)
	_, _ = fmt.Println("Done waiting")
	return w
}

func (w *When) DeleteWorkflow() *When {
	w.t.Helper()
	_, _ = fmt.Println("Deleting", w.wf.Name)
	ctx := context.Background()
	err := w.client.Delete(ctx, w.wf.Name, metav1.DeleteOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) And(block func()) *When {
	w.t.Helper()
	block()
	return w
}

func (w *When) Exec(name string, args []string, block func(t *testing.T, output string, err error)) *When {
	w.t.Helper()
	output, err := Exec(name, args...)
	block(w.t, output, err)
	return w
}

func (w *When) RunCli(args []string, block func(t *testing.T, output string, err error)) *When {
	w.t.Helper()
	return w.Exec("../../dist/argo", append([]string{"-n", Namespace}, args...), block)
}

func (w *When) CreateConfigMap(name string, data map[string]string) *When {
	w.t.Helper()

	ctx := context.Background()
	_, err := w.kubeClient.CoreV1().ConfigMaps(Namespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{Label: "true"}},
		Data:       data,
	}, metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) DeleteConfigMap(name string) *When {
	w.t.Helper()
	ctx := context.Background()
	err := w.kubeClient.CoreV1().ConfigMaps(Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) PodsQuota(podLimit int) *When {
	w.t.Helper()
	ctx := context.Background()
	list, err := w.kubeClient.CoreV1().Pods(Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	podLimit += len(list.Items)
	_, _ = fmt.Println("setting pods quota to", podLimit)
	return w.createResourceQuota("pods-quota", corev1.ResourceList{"pods": resource.MustParse(strconv.Itoa(podLimit))})
}

func (w *When) MemoryQuota(memoryLimit string) *When {
	w.t.Helper()
	return w.createResourceQuota("memory-quota", corev1.ResourceList{corev1.ResourceLimitsMemory: resource.MustParse(memoryLimit)})
}

func (w *When) StorageQuota(storageLimit string) *When {
	w.t.Helper()
	return w.createResourceQuota("storage-quota", corev1.ResourceList{"requests.storage": resource.MustParse(storageLimit)})
}

func (w *When) createResourceQuota(name string, rl corev1.ResourceList) *When {
	w.t.Helper()
	ctx := context.Background()
	_, err := w.kubeClient.CoreV1().ResourceQuotas(Namespace).Create(ctx, &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{Label: "true"}},
		Spec:       corev1.ResourceQuotaSpec{Hard: rl},
	}, metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) DeletePodsQuota() *When {
	w.t.Helper()
	return w.deleteResourceQuota("pods-quota")
}

func (w *When) DeleteStorageQuota() *When {
	w.t.Helper()
	return w.deleteResourceQuota("storage-quota")
}

func (w *When) DeleteMemoryQuota() *When {
	w.t.Helper()
	return w.deleteResourceQuota("memory-quota")
}

func (w *When) deleteResourceQuota(name string) *When {
	w.t.Helper()
	ctx := context.Background()
	err := w.kubeClient.CoreV1().ResourceQuotas(Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) Then() *Then {
	return &Then{
		t:           w.t,
		wf:          w.wf,
		cronWf:      w.cronWf,
		client:      w.client,
		cronClient:  w.cronClient,
		hydrator:    w.hydrator,
		kubeClient:  w.kubeClient,
		bearerToken: w.bearerToken,
	}
}

func (w *When) Given() *Given {
	return &Given{
		t:                 w.t,
		client:            w.client,
		wfebClient:        w.wfebClient,
		wfTemplateClient:  w.wfTemplateClient,
		cwfTemplateClient: w.cwfTemplateClient,
		cronClient:        w.cronClient,
		hydrator:          w.hydrator,
		wf:                w.wf,
		wfeb:              w.wfeb,
		wfTemplates:       w.wfTemplates,
		cwfTemplates:      w.cwfTemplates,
		cronWf:            w.cronWf,
		kubeClient:        w.kubeClient,
	}
}
