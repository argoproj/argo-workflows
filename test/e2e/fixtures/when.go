package fixtures

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"

	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v3/config"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
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
	wftsClient        v1alpha1.WorkflowTaskSetInterface
	cwfTemplateClient v1alpha1.ClusterWorkflowTemplateInterface
	cronClient        v1alpha1.CronWorkflowInterface
	hydrator          hydrator.Interface
	kubeClient        kubernetes.Interface
	bearerToken       string
	restConfig        *rest.Config
	config            *config.Config
}

func (w *When) SubmitWorkflow() *When {
	w.t.Helper()
	if w.wf == nil {
		w.t.Fatal("No workflow to submit")
	}
	_, _ = fmt.Println("Submitting workflow", w.wf.Name, w.wf.GenerateName)
	ctx := context.Background()
	label(w.wf)
	wf, err := w.client.Create(ctx, w.wf, metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.wf = wf
	}
	return w
}

func (w *When) GetWorkflow() *wfv1.Workflow {
	return w.wf
}

func label(obj metav1.Object) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	if labels[Label] == "" {
		labels[Label] = "true"
		obj.SetLabels(labels)
	}
}

func (w *When) SubmitWorkflowsFromWorkflowTemplates() *When {
	w.t.Helper()
	ctx := context.Background()
	for _, tmpl := range w.wfTemplates {
		_, _ = fmt.Println("Submitting workflow from workflow template", tmpl.Name)
		wf, err := w.client.Create(ctx, common.NewWorkflowFromWorkflowTemplate(tmpl.Name, false), metav1.CreateOptions{})
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
		wf, err := w.client.Create(ctx, common.NewWorkflowFromWorkflowTemplate(tmpl.Name, true), metav1.CreateOptions{})
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
	label(w.cronWf)
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
	label(w.wfeb)
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
		label(wfTmpl)
		if wfTmpl.Spec.WorkflowMetadata == nil {
			wfTmpl.Spec.WorkflowMetadata = &wfv1.WorkflowMetadata{Labels: map[string]string{}}
		}
		wfTmpl.Spec.WorkflowMetadata.Labels[Label] = "true"
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
		label(cwfTmpl)
		if cwfTmpl.Spec.WorkflowMetadata == nil {
			cwfTmpl.Spec.WorkflowMetadata = &wfv1.WorkflowMetadata{Labels: map[string]string{}}
		}
		cwfTmpl.Spec.WorkflowMetadata.Labels[Label] = "true"
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
	label(w.cronWf)
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
	ToHaveFailedPod Condition = func(wf *wfv1.Workflow) (bool, string) {
		return wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
			return node.Type == wfv1.NodeTypePod && node.Phase == wfv1.NodeFailed
		}), "to have failed pod"
	}
	ToBePending = ToHavePhase(wfv1.WorkflowPending)
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

var ToBeHoldingAMutex Condition = func(wf *wfv1.Workflow) (bool, string) {
	return wf.Status.Synchronization != nil && wf.Status.Synchronization.Mutex != nil && len(wf.Status.Synchronization.Mutex.Holding) > 0, "to be holding a mutex"
}

var ToBeWaitingOnASemaphore Condition = func(wf *wfv1.Workflow) (bool, string) {
	return wf.Status.Synchronization != nil && wf.Status.Synchronization.Semaphore != nil && len(wf.Status.Synchronization.Semaphore.Waiting) > 0, "to be waiting on a semaphore"
}

var ToBeHoldingASemaphore Condition = func(wf *wfv1.Workflow) (bool, string) {
	return wf.Status.Synchronization != nil && wf.Status.Synchronization.Semaphore != nil && len(wf.Status.Synchronization.Semaphore.Holding) > 0, "to be holding a semaphore"
}

type WorkflowCompletionOkay bool

func (w *When) listOptions() metav1.ListOptions {
	w.t.Helper()
	fieldSelector := ""
	if w.wf != nil {
		fieldSelector = "metadata.name=" + w.wf.Name
	}

	labelSelector := Label
	if w.cronWf != nil {
		labelSelector += "," + common.LabelKeyCronWorkflow + "=" + w.cronWf.Name
	}
	return metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector}
}

func describeListOptions(opts metav1.ListOptions) string {
	out := ""
	if opts.FieldSelector != "" {
		out += fmt.Sprintf("field selector '%s'", opts.FieldSelector)
		if opts.LabelSelector != "" {
			out += " and "
		}
	}
	if opts.LabelSelector != "" {
		out += fmt.Sprintf("label selector '%s'", opts.LabelSelector)
	}
	return out
}

// Wait for a workflow to meet a condition:
// Options:
// * `time.Duration` - change the timeout - 30s by default
//
// * `metav1.ListOptions` - override label/field selectors
//
// * `WorkflowCompletionOkay" (bool alias): if this is true, we won't stop checking for the other options
//   - just because the Workflow completed
//
// * `Condition` - a condition - `ToFinish` by default
func (w *When) WaitForWorkflow(options ...interface{}) *When {
	w.t.Helper()
	timeout := defaultTimeout
	condition := ToBeDone
	listOptions := w.listOptions()
	var workflowCompletionOkay WorkflowCompletionOkay
	for _, opt := range options {
		switch v := opt.(type) {
		case time.Duration:
			// Note that we add the timeoutBias (defaults to 0), set by environment variable E2E_WAIT_TIMEOUT_BIAS
			timeout = v + timeoutBias
		case metav1.ListOptions:
			listOptions = v
		case Condition:
			condition = v
		case WorkflowCompletionOkay:
			workflowCompletionOkay = v
		default:
			w.t.Fatal("unknown option type: " + reflect.TypeOf(opt).String())
		}
	}

	start := time.Now()
	_, _ = fmt.Printf("Waiting up to %s for workflow with %s\n", timeout, describeListOptions(listOptions))

	ctx := context.Background()

	watch, err := w.client.Watch(ctx, listOptions)
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
				w.hydrateWorkflow(wf)
				printWorkflow(wf)
				if ok, message := condition(wf); ok {
					_, _ = fmt.Printf("Condition %q met after %s\n", message, time.Since(start).Truncate(time.Second))
					w.wf = wf
					return w
				}
				if !workflowCompletionOkay {
					// once done the workflow is done, the condition can never be met
					// rather than wait maybe 30s for something that can never happen
					if ok, _ = ToBeDone(wf); ok {
						w.t.Errorf("condition never and cannot be met because the workflow is done")
						return w
					}
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

// Waits for workflow to be created with different name than the current one
func (w *When) WaitForNewWorkflow(condition Condition) *When {
	w.t.Helper()
	if w.wf == nil {
		w.t.Fatal("No previous workflow")
	}
	listOptions := w.listOptions()
	listOptions.FieldSelector = "metadata.name!=" + w.wf.Name
	w.wf = nil
	return w.WaitForWorkflow(condition, WorkflowCompletionOkay(true), listOptions)
}

func (w *When) WaitForWorkflowListCount(timeout time.Duration, count int) *When {
	w.t.Helper()
	return w.waitForWorkflowListCount(timeout+timeoutBias, w.listOptions().LabelSelector, count)
}

func (w *When) waitForWorkflowListCount(timeout time.Duration, labelSelector string, count int) *When {
	w.t.Helper()

	start := time.Now()
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	_, _ = fmt.Printf("Waiting up to %s for %d workflows with %s\n", timeout, count, describeListOptions(opts))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			w.t.Errorf("timeout after %v waiting for condition", timeout)
			return w
		default:
			wfList, err := w.client.List(ctx, opts)
			if err != nil {
				w.t.Error(err)
				return w
			}
			if len(wfList.Items) == count {
				_, _ = fmt.Printf("Condition met after %s\n", time.Since(start).Truncate(time.Second))
				return w
			}
		}
		time.Sleep(time.Second)
	}
}

func (w *When) WaitForWorkflowDeletion() *When {
	w.t.Helper()
	return w.WaitForWorkflowListCount(defaultTimeout, 0)
}

func (w *When) WaitForWorkflowListFailedCount(count int) *When {
	w.t.Helper()
	return w.waitForWorkflowListCount(defaultTimeout, Label+","+common.LabelKeyPhase+"=Failed", count)
}

func (w *When) WaitForCronWorkflowCompleted(timeout time.Duration) *When {
	w.t.Helper()
	return w.waitForCronWorkflow(timeout+timeoutBias, Label+","+common.LabelKeyCronWorkflowCompleted+"=true", false)
}

func (w *When) WaitForCronWorkflow() *When {
	w.t.Helper()
	return w.waitForCronWorkflow(defaultTimeout, Label, true)
}

func (w *When) waitForCronWorkflow(timeout time.Duration, labelSelector string, onlyActive bool) *When {
	w.t.Helper()
	if w.cronWf == nil {
		w.t.Fatal("No cron workflow")
	}
	fieldSelector := "metadata.name=" + w.cronWf.Name
	opts := metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector}
	start := time.Now()
	_, _ = fmt.Printf("Waiting up to %s for cron workflow with %s\n", timeout, describeListOptions(opts))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			w.t.Errorf("timeout after %v waiting for condition", timeout)
			return w
		default:
			cronWfList, err := w.cronClient.List(ctx, opts)
			if err != nil {
				w.t.Error(err)
				return w
			}
			if len(cronWfList.Items) == 1 && (!onlyActive || len(cronWfList.Items[0].Status.Active) == 1) {
				_, _ = fmt.Printf("Condition met after %s\n", time.Since(start).Truncate(time.Second))
				return w
			}
		}
		time.Sleep(time.Second)
	}

}

func (w *When) hydrateWorkflow(wf *wfv1.Workflow) {
	w.t.Helper()
	err := w.hydrator.Hydrate(wf)
	if err != nil {
		w.t.Fatal(err)
	}
}

// Wait creates slow flaky tests
// DEPRECATED: do not use this
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

func (w *When) RemoveFinalizers(shouldErr bool) *When {
	w.t.Helper()
	ctx := context.Background()

	_, err := w.client.Patch(ctx, w.wf.Name, types.MergePatchType, []byte("{\"metadata\":{\"finalizers\":null}}"), metav1.PatchOptions{})
	if err != nil && shouldErr {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) AddNamespaceLimit(limit string) *When {
	w.t.Helper()
	ctx := context.Background()
	patchMap := make(map[string]interface{})
	metadata := make(map[string]interface{})
	labels := make(map[string]interface{})
	labels["workflows.argoproj.io/parallelism-limit"] = limit
	metadata["labels"] = labels
	patchMap["metadata"] = metadata

	bs, err := json.Marshal(patchMap)
	if err != nil {
		w.t.Fatal(err)
	}

	_, err = w.kubeClient.CoreV1().Namespaces().Patch(ctx, Namespace, types.MergePatchType, []byte(bs), metav1.PatchOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

type PodCondition func(p *corev1.Pod) bool

var (
	PodCompleted PodCondition = func(p *corev1.Pod) bool {
		return p.Labels[common.LabelKeyCompleted] == "true"
	}
	PodDeleted PodCondition = func(p *corev1.Pod) bool {
		return !p.DeletionTimestamp.IsZero()
	}
)

func (w *When) WaitForPod(condition PodCondition) *When {
	w.t.Helper()
	ctx := context.Background()
	timeout := defaultTimeout
	watch, err := w.kubeClient.CoreV1().Pods(Namespace).Watch(
		ctx,
		metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + w.wf.Name, TimeoutSeconds: ptr.To(int64(timeout.Seconds()))},
	)
	if err != nil {
		w.t.Fatal(err)
	}
	defer watch.Stop()
	for event := range watch.ResultChan() {
		p := event.Object.(*corev1.Pod)
		state := p.Status.Phase
		if p.Labels[common.LabelKeyCompleted] == "true" {
			state = "Complete"
		}
		if !p.DeletionTimestamp.IsZero() {
			state = "Deleted"
		}
		_, _ = fmt.Printf("pod %s: %s\n", p.Name, state)
		if condition(p) {
			_, _ = fmt.Printf("Pod condition met\n")
			return w
		}
	}
	w.t.Fatal(fmt.Errorf("timeout after %v waiting for pod", timeout))
	return w
}

func (w *When) And(block func()) *When {
	w.t.Helper()
	block()
	return w
}

func (w *When) Exec(name string, args []string, block func(t *testing.T, output string, err error)) *When {
	w.t.Helper()
	output, err := Exec(name, "", args...)
	block(w.t, output, err)
	return w
}

func (w *When) RunCli(args []string, block func(t *testing.T, output string, err error)) *When {
	w.t.Helper()
	if !strings.HasPrefix(w.t.Name(), "TestCLISuite/") {
		w.t.Fatal("You cannot use RunCli for tests that are not in TestCLISuite")
	}
	return w.Exec("../../dist/argo", append([]string{"-n", Namespace}, args...), block)
}

func (w *When) CreateConfigMap(name string, data map[string]string, customLabels map[string]string) *When {
	w.t.Helper()

	labels := map[string]string{Label: "true"}

	for k, v := range customLabels {
		labels[k] = v
	}

	ctx := context.Background()
	_, err := w.kubeClient.CoreV1().ConfigMaps(Namespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		Data:       data,
	}, metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) UpdateConfigMap(name string, data map[string]string, customLabels map[string]string) *When {
	w.t.Helper()

	labels := map[string]string{Label: "true"}

	for k, v := range customLabels {
		labels[k] = v
	}

	ctx := context.Background()
	_, err := w.kubeClient.CoreV1().ConfigMaps(Namespace).Update(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		Data:       data,
	}, metav1.UpdateOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) DeleteConfigMap(name string) *When {
	w.t.Helper()
	ctx := context.Background()
	fmt.Printf("deleting configmap %s\n", name)
	err := w.kubeClient.CoreV1().ConfigMaps(Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

// setupDBSession creates a database session for semaphore operations
// The caller must defer closing the returned session
func (w *When) setupDBSession() db.Session {
	w.t.Helper()
	// Skip if persistence is not enabled
	if w.config == nil || w.config.Synchronization == nil {
		w.t.Fatal("Require synchronization to be setup")
	}

	ctx := context.Background()
	dbSession, err := sqldb.CreateDBSession(ctx, w.kubeClient, Namespace, w.config.Synchronization.DBConfig)
	if err != nil {
		w.t.Fatal(err)
	}
	return dbSession
}

// SetupDatabaseSemaphore creates a database semaphore with the specified limit
func (w *When) SetupDatabaseSemaphore(name string, limit int) *When {
	dbSession := w.setupDBSession()
	defer dbSession.Close()

	// Get the table name from config
	limitTable := w.config.Synchronization.LimitTableName

	// Insert or update the semaphore limit
	if w.config.Synchronization.PostgreSQL != nil {
		_, err := dbSession.SQL().Exec(
			fmt.Sprintf("INSERT INTO %s (name, sizelimit) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET sizelimit = $2",
				limitTable),
			name, limit)
		if err != nil {
			w.t.Fatal(err)
		}
	} else if w.config.Synchronization.MySQL != nil {
		_, err := dbSession.SQL().Exec(
			fmt.Sprintf("INSERT INTO %s (name, sizelimit) VALUES (?, ?) ON DUPLICATE KEY UPDATE sizelimit = ?",
				limitTable),
			name, limit, limit)
		if err != nil {
			w.t.Fatal(err)
		}
	} else {
		w.t.Fatal("Require one synchronization database to be setup")
	}
	return w
}

// SetDBSemaphoreState inserts or updates a record in the semaphore state table
func (w *When) SetDBSemaphoreState(name string, workflowKey string, controller *string, mutex bool, held bool, priority int32, timestamp time.Time) *When {
	dbSession := w.setupDBSession()
	defer dbSession.Close()

	// Get the table name from config
	stateTable := w.config.Synchronization.StateTableName

	controllerName := w.config.Synchronization.ControllerName
	if controller != nil {
		controllerName = *controller
	}

	dbKey := "sem/" + name
	if mutex {
		dbKey = "mtx/" + name
	}

	// Insert or update the semaphore state
	if w.config.Synchronization.PostgreSQL != nil {
		_, err := dbSession.SQL().Exec(
			fmt.Sprintf("INSERT INTO %s (name, workflowkey, controller, held, priority, time) VALUES ($1, $2, $3, $4, $5, $6) "+
				"ON CONFLICT (name, workflowkey, controller) DO UPDATE SET held = $4, priority = $5, time = $6",
				stateTable),
			dbKey, workflowKey, controllerName, held, priority, timestamp)
		if err != nil {
			w.t.Fatal(err)
		}
	} else if w.config.Synchronization.MySQL != nil {
		_, err := dbSession.SQL().Exec(
			fmt.Sprintf("INSERT INTO %s (name, workflowkey, controller, held, priority, time) VALUES (?, ?, ?, ?, ?, ?) "+
				"ON DUPLICATE KEY UPDATE held = ?, priority = ?, time = ?",
				stateTable),
			dbKey, workflowKey, controllerName, held, priority, timestamp, held, priority, timestamp)
		if err != nil {
			w.t.Fatal(err)
		}
	} else {
		w.t.Fatal("Require one synchronization database to be setup")
	}
	return w
}

// SetDBSemaphoreControllerHB inserts or updates a record in the semaphore controller heartbeat table
func (w *When) SetDBSemaphoreControllerHB(name *string, timestamp time.Time) *When {
	dbSession := w.setupDBSession()
	defer dbSession.Close()

	// Get the table name from config
	controllerTable := w.config.Synchronization.ControllerTableName

	controllerName := w.config.Synchronization.ControllerName
	if name != nil {
		controllerName = *name
	}

	// Insert or update the semaphore state
	if w.config.Synchronization.PostgreSQL != nil {
		_, err := dbSession.SQL().Exec(
			fmt.Sprintf("INSERT INTO %s (controller, time) VALUES ($1, $2) "+
				"ON CONFLICT (controller) DO UPDATE SET time = $2",
				controllerTable),
			controllerName, timestamp)
		if err != nil {
			w.t.Fatal(err)
		}
	} else if w.config.Synchronization.MySQL != nil {
		_, err := dbSession.SQL().Exec(
			fmt.Sprintf("INSERT INTO %s (controller, time) VALUES (?, ?) "+
				"ON DUPLICATE KEY UPDATE time = ?",
				controllerTable),
			controllerName, timestamp, timestamp)
		if err != nil {
			w.t.Fatal(err)
		}
	} else {
		w.t.Fatal("Require one synchronization database to be setup")
	}
	return w
}

// ClearDBSemaphoreState deletes all records from the semaphore state table except for the controller heartbeat records
func (w *When) ClearDBSemaphoreState() *When {
	w.t.Helper()
	dbSession := w.setupDBSession()
	defer dbSession.Close()

	_, err := dbSession.SQL().Exec(
		fmt.Sprintf("DELETE FROM %s", w.config.Synchronization.StateTableName),
	)
	if err != nil {
		w.t.Fatal(err)
	}
	// Delete all records except for this controller heartbeat records
	_, err = dbSession.SQL().Exec(
		fmt.Sprintf("DELETE FROM %s WHERE controller != ?", w.config.Synchronization.ControllerTableName),
		w.config.Synchronization.ControllerName,
	)
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

func (w *When) ResumeCronWorkflow(string) *When {
	w.t.Helper()
	return w.setCronWorkflowSuspend(false)
}

func (w *When) SuspendCronWorkflow() *When {
	w.t.Helper()
	return w.setCronWorkflowSuspend(true)
}

func (w *When) setCronWorkflowSuspend(suspend bool) *When {
	ctx := context.Background()
	w.t.Helper()
	spec := map[string]interface{}{"suspend": suspend}
	data, err := json.Marshal(map[string]interface{}{"spec": spec})
	if err != nil {
		w.t.Fatal(err)
	}
	_, err = w.cronClient.Patch(ctx, w.cronWf.Name, types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) ShutdownWorkflow(strategy wfv1.ShutdownStrategy) *When {
	w.t.Helper()
	ctx := context.Background()
	data, err := json.Marshal(map[string]interface{}{"spec": map[string]interface{}{"shutdown": strategy}})
	if err != nil {
		w.t.Fatal(err)
	}
	_, err = w.client.Patch(ctx, w.wf.Name, types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

// test condition function on Workflow
func (w *When) WorkflowCondition(condition func(wf *wfv1.Workflow) bool) bool {
	return condition(w.wf)
}

func (w *When) Then() *Then {
	return &Then{
		t:           w.t,
		wf:          w.wf,
		cronWf:      w.cronWf,
		client:      w.client,
		wftsClient:  w.wftsClient,
		cronClient:  w.cronClient,
		hydrator:    w.hydrator,
		kubeClient:  w.kubeClient,
		bearerToken: w.bearerToken,
		restConfig:  w.restConfig,
	}
}

func (w *When) Given() *Given {
	return &Given{
		t:                 w.t,
		client:            w.client,
		wfebClient:        w.wfebClient,
		wfTemplateClient:  w.wfTemplateClient,
		wftsClient:        w.wftsClient,
		cwfTemplateClient: w.cwfTemplateClient,
		cronClient:        w.cronClient,
		hydrator:          w.hydrator,
		wf:                w.wf,
		wfeb:              w.wfeb,
		wfTemplates:       w.wfTemplates,
		cwfTemplates:      w.cwfTemplates,
		cronWf:            w.cronWf,
		kubeClient:        w.kubeClient,
		restConfig:        w.restConfig,
		config:            w.config,
	}
}
