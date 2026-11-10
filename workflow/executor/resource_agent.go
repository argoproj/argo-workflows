package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	argoerr "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// resourceTaskSource yields the resource tasks fed to it, each node once,
// to every resource worker's harness.
type resourceTaskSource struct {
	tasks      chan Task
	considered sync.Map
}

var _ TaskSource = &resourceTaskSource{}

func (s *resourceTaskSource) Next(ctx context.Context) (Task, bool, error) {
	for {
		select {
		case <-ctx.Done():
			return Task{}, false, nil
		case task, ok := <-s.tasks:
			if !ok {
				return Task{}, false, nil
			}
			if _, seen := s.considered.LoadOrStore(task.Config.NodeID, true); seen {
				continue
			}
			return task, true, nil
		}
	}
}

// feed sends the task set's resource tasks to the workers, until ctx ends.
func (s *resourceTaskSource) feed(ctx context.Context, workflowName string, workflowUID types.UID, taskSet *wfv1.WorkflowTaskSet) {
	for nodeID, tmpl := range taskSet.Spec.Tasks {
		if tmpl.Resource == nil {
			continue
		}
		task := Task{Config: TaskConfig{WorkflowName: workflowName, WorkflowUID: workflowUID, NodeID: nodeID, Template: tmpl}}
		select {
		case <-ctx.Done():
			return
		case s.tasks <- task:
		}
	}
}

// resourceSink sends a task's result to the patch worker, which writes it
// to the task set's status with the other tasks' results.
type resourceSink struct {
	nodeID    string
	responses chan<- response
}

var _ ResultSink = resourceSink{}

func (resourceSink) Start(context.Context) error { return nil }

func (s resourceSink) Report(ctx context.Context, r wfv1.NodeResult) error {
	if r.Phase == "" {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.responses <- response{NodeID: s.nodeID, Result: &r}:
		return nil
	}
}

func (resourceSink) Progress(context.Context, wfv1.Progress) error { return nil }

func (s resourceSink) Fail(ctx context.Context, err error) {
	_ = s.Report(ctx, wfv1.NodeResult{Phase: wfv1.NodeError, Message: fmt.Sprintf("error processing task: %s", err)})
}

func (resourceSink) Finalize(context.Context) error { return nil }

type informerKey struct {
	gvr       schema.GroupVersionResource
	namespace string
}

// resourceState is what an informer last saw of a task's resource.
type resourceState struct {
	obj     *unstructured.Unstructured
	deleted bool
}

// expectedResource is a task whose resource is yet to meet a condition.
type expectedResource struct {
	template    *wfv1.Template
	successReqs labels.Requirements
	failReqs    labels.Requirements
}

// resourceWatch watches the resources the tasks claim, with one informer
// per resource type and namespace that outlives the task that started it.
// An informer handler stores the resource's state by node ID and sends the
// node ID on outputs.
type resourceWatch struct {
	client      dynamic.Interface
	workflowUID string
	stop        <-chan struct{}
	outputs     chan string
	// states holds the last observed resourceState per node ID.
	states sync.Map
	// expected holds the expectedResource per node ID until its result.
	expected  sync.Map
	mu        sync.Mutex
	informers map[informerKey]cache.SharedIndexInformer
}

func newResourceWatch(client dynamic.Interface, workflowUID string) *resourceWatch {
	return &resourceWatch{
		client:      client,
		workflowUID: workflowUID,
		outputs:     make(chan string),
		informers:   map[informerKey]cache.SharedIndexInformer{},
	}
}

// expect registers the task whose resource the informers are to report on.
func (w *resourceWatch) expect(nodeID string, tmpl *wfv1.Template) error {
	successReqs, err := parseResourceCondition("success", tmpl.Resource.SuccessCondition)
	if err != nil {
		return err
	}
	failReqs, err := parseResourceCondition("fail", tmpl.Resource.FailureCondition)
	if err != nil {
		return err
	}
	w.expected.Store(nodeID, expectedResource{template: tmpl, successReqs: successReqs, failReqs: failReqs})
	return nil
}

func parseResourceCondition(kind, condition string) (labels.Requirements, error) {
	if condition == "" {
		return nil, nil
	}
	selector, err := labels.Parse(condition)
	if err != nil {
		return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "%s condition '%s' failed to parse: %v", kind, condition, err)
	}
	reqs, _ := selector.Requirements()
	return reqs, nil
}

// claim labels obj as the workflow's and annotates it with the node ID, so
// that the informers list it and attribute its events to the node.
func (w *resourceWatch) claim(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured, nodeID string) error {
	patch, err := json.Marshal(map[string]any{"metadata": map[string]any{
		"labels":      map[string]string{common.LabelKeyAgentResource: w.workflowUID},
		"annotations": map[string]string{common.AnnotationKeyNodeID: nodeID},
	}})
	if err != nil {
		return err
	}
	_, err = w.client.Resource(gvr).Namespace(obj.GetNamespace()).Patch(ctx, obj.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}

// ensure runs an informer for the resource type in the namespace, unless
// one runs already, and waits for its first list.
func (w *resourceWatch) ensure(ctx context.Context, gvr schema.GroupVersionResource, namespace string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	key := informerKey{gvr: gvr, namespace: namespace}
	if informer, ok := w.informers[key]; ok {
		return w.waitSynced(ctx, gvr, informer)
	}
	informer := dynamicinformer.NewFilteredDynamicInformer(w.client, gvr, namespace, 0, cache.Indexers{}, func(opts *metav1.ListOptions) {
		opts.LabelSelector = common.LabelKeyAgentResource + "=" + w.workflowUID
	}).Informer()
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(o any) { w.observe(o, false) },
		UpdateFunc: func(_, o any) { w.observe(o, false) },
		DeleteFunc: func(o any) { w.observe(o, true) },
	})
	if err != nil {
		return err
	}
	go informer.Run(w.stop)
	w.informers[key] = informer
	return w.waitSynced(ctx, gvr, informer)
}

// theory-debt: the sync timeout is the only signal that the agent may not list or watch the resource type.
func (w *resourceWatch) waitSynced(ctx context.Context, gvr schema.GroupVersionResource, informer cache.SharedIndexInformer) error {
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !cache.WaitForCacheSync(syncCtx.Done(), informer.HasSynced) {
		return fmt.Errorf("timed out waiting to list and watch %s", gvr)
	}
	return nil
}

// observe records the resource's state and sends its node ID on outputs.
func (w *resourceWatch) observe(o any, deleted bool) {
	if tombstone, ok := o.(cache.DeletedFinalStateUnknown); ok {
		o = tombstone.Obj
	}
	obj, ok := o.(*unstructured.Unstructured)
	if !ok {
		return
	}
	nodeID := obj.GetAnnotations()[common.AnnotationKeyNodeID]
	if nodeID == "" {
		return
	}
	w.states.Store(nodeID, resourceState{obj: obj, deleted: deleted})
	select {
	case <-w.stop:
	case w.outputs <- nodeID:
	}
}

// evaluate returns the node's result if its resource has reached one. A
// node that is not expected, or whose state is unknown, has none.
func (w *resourceWatch) evaluate(ctx context.Context, nodeID string) (wfv1.NodeResult, bool) {
	e, ok := w.expected.Load(nodeID)
	if !ok {
		return wfv1.NodeResult{}, false
	}
	s, ok := w.states.Load(nodeID)
	if !ok {
		return wfv1.NodeResult{}, false
	}
	result, done := resourceResult(ctx, e.(expectedResource), s.(resourceState))
	if done {
		w.expected.Delete(nodeID)
		w.states.Delete(nodeID)
	}
	return result, done
}

func resourceResult(ctx context.Context, expected expectedResource, state resourceState) (wfv1.NodeResult, bool) {
	jsonBytes, err := state.obj.MarshalJSON()
	if err != nil {
		return wfv1.NodeResult{Phase: wfv1.NodeError, Message: err.Error()}, true
	}
	waiting, condErr := matchConditions(ctx, jsonBytes, expected.successReqs, expected.failReqs)
	switch {
	case condErr == nil:
		outputs, err := resourceOutputs(ctx, expected.template, state.obj)
		if err != nil {
			return wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: err.Error()}, true
		}
		return wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: outputs}, true
	case !waiting:
		return wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: condErr.Error()}, true
	case state.deleted:
		return wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: "The resource has been deleted while its status was still being checked."}, true
	}
	return wfv1.NodeResult{}, false
}

// resourceOutputs resolves the template's output parameters against obj, or
// to their defaults when there is no object.
func resourceOutputs(ctx context.Context, tmpl *wfv1.Template, obj *unstructured.Unstructured) (*wfv1.Outputs, error) {
	outputs := tmpl.Outputs.DeepCopy()
	var jsonBytes []byte
	if obj != nil && slices.ContainsFunc(outputs.Parameters, needsJQRead) {
		withoutManagedFields := obj.DeepCopy()
		withoutManagedFields.SetManagedFields(nil)
		var err error
		if jsonBytes, err = withoutManagedFields.MarshalJSON(); err != nil {
			return nil, err
		}
	}
	for i, param := range outputs.Parameters {
		if param.ValueFrom == nil {
			continue
		}
		if obj == nil {
			value := ""
			if param.ValueFrom.Default != nil {
				value = param.ValueFrom.Default.String()
			}
			outputs.Parameters[i].Value = wfv1.AnyStringPtr(value)
			continue
		}
		value, ok, err := resourceParameterValue(ctx, param, obj, jsonBytes)
		if err != nil {
			return nil, err
		}
		if ok {
			outputs.Parameters[i].Value = wfv1.AnyStringPtr(value)
		}
	}
	return outputs, nil
}

// runResourceTask issues the task's request. For a delete, or a get that
// finds nothing, that is the task's whole result; otherwise it claims the
// resource for the informers to report on, and the task is Running.
func (we *WorkflowExecutor) runResourceTask(ctx context.Context) error {
	resource := we.Template.Resource
	if resource.Action == "delete" && (resource.SuccessCondition != "" || resource.FailureCondition != "" || len(we.Template.Outputs.Parameters) > 0) {
		return errors.New("successCondition, failureCondition and outputs are not supported for delete action")
	}
	// theory-debt: the manifest is staged at one fixed path, so the tasks of one process must not overlap.
	if err := we.StageFiles(ctx); err != nil {
		return err
	}
	obj, err := we.execResource(ctx, resource.Action, we.resourceManifestPath(), resource.Flags)
	if err != nil {
		return err
	}
	if obj == nil {
		outputs, err := resourceOutputs(ctx, &we.Template, nil)
		if err != nil {
			return err
		}
		return we.sink.Report(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: outputs})
	}
	if err := we.resourceWatch.expect(we.nodeID, &we.Template); err != nil {
		return err
	}
	gvr := inferGVR(*obj)
	if err := we.resourceWatch.claim(ctx, gvr, obj, we.nodeID); err != nil {
		return err
	}
	if err := we.resourceWatch.ensure(ctx, gvr, obj.GetNamespace()); err != nil {
		return err
	}
	return we.sink.Report(ctx, wfv1.NodeResult{Phase: wfv1.NodeRunning})
}

func (we *WorkflowExecutor) resourceManifestPath() string {
	from := we.Template.Resource.ManifestFrom
	if from == nil {
		return common.ExecutorResourceManifestPath
	}
	for _, art := range we.Template.Inputs.Artifacts {
		if art.Name == from.Artifact.Name {
			return art.Path
		}
	}
	return common.ExecutorResourceManifestPath
}

// ResourceAgentExecutor runs a workflow's resource tasks. Each task issues
// its request and claims its resource; the informers, which outlive the
// tasks, carry the results to the task set.
type ResourceAgentExecutor struct {
	WorkflowName      string
	workflowUID       string
	WorkflowInterface workflow.Interface
	process           *Process
	watch             *resourceWatch
	taskWorkers       int
	requeueTime       time.Duration
}

// NewResourceAgentExecutor gives process the watch its resource tasks
// claim resources with.
func NewResourceAgentExecutor(process *Process, wfClient workflow.Interface, dynamicClient dynamic.Interface, workflowName, workflowUID string, taskWorkers int, requeueTime time.Duration) *ResourceAgentExecutor {
	watch := newResourceWatch(dynamicClient, workflowUID)
	process.resourceWatch = watch
	return &ResourceAgentExecutor{
		WorkflowName:      workflowName,
		workflowUID:       workflowUID,
		WorkflowInterface: wfClient,
		process:           process,
		watch:             watch,
		taskWorkers:       taskWorkers,
		requeueTime:       requeueTime,
	}
}

// Run runs the task set's resource tasks until the task set is deleted or
// the workflow completes.
func (re *ResourceAgentExecutor) Run(ctx context.Context) error {
	defer runtimeutil.HandleCrashWithContext(ctx, runtimeutil.PanicHandlers...)

	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("taskWorkers", re.taskWorkers).
		WithField("requeueTime", re.requeueTime).
		Info(ctx, "Starting resource agent")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	re.watch.stop = ctx.Done()

	source := &resourceTaskSource{tasks: make(chan Task)}
	responses := make(chan response)
	taskSets := re.WorkflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(re.process.Namespace)

	go patchTaskSetStatus(ctx, taskSets, re.WorkflowName, responses, re.requeueTime)
	go re.reportResults(ctx, responses)
	for range re.taskWorkers {
		h := &Harness{
			Process: re.process,
			Source:  source,
			Sink:    func(task Task) ResultSink { return resourceSink{nodeID: task.Config.NodeID, responses: responses} },
			Role:    ResourceRole,
		}
		go func() { _ = h.Run(ctx) }()
	}
	return watchTaskSet(ctx, taskSets, re.WorkflowName, func(taskSet *wfv1.WorkflowTaskSet) {
		source.feed(ctx, re.WorkflowName, types.UID(re.workflowUID), taskSet)
	})
}

// reportResults sends each resource's result to the patch worker once the
// informers have observed it.
func (re *ResourceAgentExecutor) reportResults(ctx context.Context, responses chan<- response) {
	for {
		select {
		case <-ctx.Done():
			return
		case nodeID := <-re.watch.outputs:
			if result, done := re.watch.evaluate(ctx, nodeID); done {
				_ = resourceSink{nodeID: nodeID, responses: responses}.Report(ctx, result)
			}
		}
	}
}

// watchTaskSet calls onTaskSet with each version of the workflow's task
// set until the task set is deleted or the workflow completes.
func watchTaskSet(ctx context.Context, taskSets v1alpha1.WorkflowTaskSetInterface, name string, onTaskSet func(*wfv1.WorkflowTaskSet)) error {
	logger := logging.RequireLoggerFromContext(ctx)
	for {
		taskSetWatch, err := taskSets.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
		if err != nil {
			return err
		}
		for event := range taskSetWatch.ResultChan() {
			logger.WithField("event_type", event.Type).Info(ctx, "TaskSet Event")
			if event.Type == watch.Deleted {
				return nil
			}
			taskSet, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			if IsWorkflowCompleted(taskSet) {
				logger.Info(ctx, "Workflow completed... stopping")
				return nil
			}
			onTaskSet(taskSet)
		}
	}
}

// patchTaskSetStatus patches the results it receives into the task set's
// status, batched per requeueTime.
func patchTaskSetStatus(ctx context.Context, taskSets v1alpha1.WorkflowTaskSetInterface, name string, responses <-chan response, requeueTime time.Duration) {
	ticker := time.NewTicker(requeueTime)
	defer ticker.Stop()
	nodeResults := map[string]wfv1.NodeResult{}
	logger := logging.RequireLoggerFromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-responses:
			nodeResults[res.NodeID] = *res.Result
		case <-ticker.C:
			if len(nodeResults) == 0 {
				continue
			}
			patch, err := json.Marshal(map[string]any{"status": wfv1.WorkflowTaskSetStatus{Nodes: nodeResults}})
			if err != nil {
				logger.WithError(err).Error(ctx, "Generating Patch Failed")
				continue
			}
			err = retry.OnError(wait.Backoff{
				Duration: time.Second,
				Factor:   2,
				Jitter:   0.1,
				Steps:    5,
				Cap:      30 * time.Second,
			}, func(retryErr error) bool {
				return argoerr.IsTransientErr(ctx, retryErr)
			}, func() error {
				_, patchErr := taskSets.Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{}, "status")
				return patchErr
			})
			if err != nil && !argoerr.IsTransientErr(ctx, err) {
				logger.WithError(err).Error(ctx, "TaskSet Patch Failed")
				for node := range nodeResults {
					nodeResults[node] = wfv1.NodeResult{
						Phase:   wfv1.NodeError,
						Message: fmt.Sprintf("task completed but an error occurred when patching its result: %s", err),
					}
				}
				continue
			}
			nodeResults = map[string]wfv1.NodeResult{}
			logger.Info(ctx, "Patched TaskSet")
		}
	}
}
