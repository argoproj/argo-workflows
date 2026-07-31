package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	kubectlget "k8s.io/kubectl/pkg/cmd/get"
	"sigs.k8s.io/yaml"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
	apiworkflow "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions"
	"github.com/argoproj/argo-workflows/v4/util/env"
	argoerr "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

type informerKey struct {
	gvr       schema.GroupVersionResource
	namespace string
}

type resourceEvent struct {
	// obj is the object as the informer observed it (the last-known state for a deletion). Carrying
	// it on the event — rather than re-reading the informer store when the event is processed — is
	// what lets a resource that met its success condition and was then deleted still report
	// Succeeded: a fresh store read on the delete event would find the object already gone.
	obj *unstructured.Unstructured
	// deleted marks a delete event. If obj had not already met its success/failure condition, the
	// resource vanished mid-check and the node fails.
	deleted bool
	// nodeID (from the object's node-id annotation) attributes the event to its node.
	nodeID string
}

type resourceConditions struct {
	successReqs labels.Requirements
	failReqs    labels.Requirements
}

// ResourceAgentExecutor holds state for all resource agent work
type ResourceAgentExecutor struct {
	WorkflowName      string
	WorkflowUID       string
	ClientSet         kubernetes.Interface
	DynamicClient     dynamic.Interface
	WorkflowInterface workflow.Interface
	RESTMapper        meta.ResettableRESTMapper
	Namespace         string

	tasksMutex sync.RWMutex
	tasks      map[string]*wfv1.Template // keyed by node ID

	informersMutex sync.Mutex
	informers      map[informerKey]cache.SharedIndexInformer

	eventQueue workqueue.TypedInterface[resourceEvent]

	// resultsMutex guards pending and reported. report() writes here without blocking; patchWorker
	// drains pending on its tick. reported dedups so each node is reported at most once.
	resultsMutex sync.Mutex
	pending      map[string]wfv1.NodeResult
	reported     map[string]bool
}

// NewResourceAgentExecutor returns a new ResourceAgentExecutor
func NewResourceAgentExecutor(clientSet kubernetes.Interface, config *rest.Config, namespace, workflowName, workflowUID string) *ResourceAgentExecutor {
	return &ResourceAgentExecutor{
		WorkflowName:      workflowName,
		WorkflowUID:       workflowUID,
		ClientSet:         clientSet,
		DynamicClient:     dynamic.NewForConfigOrDie(config),
		WorkflowInterface: workflow.NewForConfigOrDie(config),
		// discovery-backed mapper resolves the real resource (plural) for any kind, including CRDs
		RESTMapper: restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(clientSet.Discovery())),
		Namespace:  namespace,
		tasks:      map[string]*wfv1.Template{},
		informers:  map[informerKey]cache.SharedIndexInformer{},
		eventQueue: workqueue.NewTyped[resourceEvent](),
		pending:    map[string]wfv1.NodeResult{},
		reported:   map[string]bool{},
	}
}

func (rae *ResourceAgentExecutor) createAndWatchResource(ctx context.Context, tmpl *wfv1.Template, nodeID string) {
	// This agent and the HTTP/plugin AgentExecutor share one WorkflowTaskSet and partition it by
	// template type: this agent serves only resource tasks, and AgentExecutor.processTask skips them
	// (its `case tmpl.Resource != nil`). The two skips must stay complementary — a new agent-served
	// template type needs matching skips on both sides or the node errors spuriously.
	if tmpl == nil || tmpl.Resource == nil {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx)
	// fail fast: don't create a resource whose conditions can never be evaluated
	rc, err := parseConditions(tmpl.Resource.SuccessCondition, tmpl.Resource.FailureCondition)
	if err != nil {
		logger.WithError(err).Error(ctx, "invalid resource conditions")
		rae.report(nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: err.Error()})
		return
	}
	manifest, err := rae.resolveManifest(ctx, tmpl)
	if err != nil {
		logger.WithError(err).Error(ctx, "failed to resolve resource manifest")
		rae.report(nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: fmt.Sprintf("failed to resolve manifest: %v", err)})
		return
	}
	obj, gvr, namespaced, err := rae.createResource(ctx, tmpl.Resource, manifest, nodeID)
	if err != nil {
		logger.WithError(err).Error(ctx, "failed to create resource")
		rae.report(nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: fmt.Sprintf("failed to create resource: %v", err)})
		return
	}
	logger.WithFields(logging.Fields{"namespace": obj.GetNamespace(), "name": obj.GetName(), "kind": obj.GetKind()}).Info(ctx, "Created resource")

	// A delete leaves no live object to watch, and genuinely-absent conditions mean there is nothing
	// to wait for: succeed immediately off the create response, as WaitResource does. Match on the raw
	// strings, not the parsed requirement counts: a whitespace-only condition (e.g. from a parameter
	// substitution) is non-empty but parses to zero requirements, and WaitResource still reads it back
	// once — so it must fall through to the poll below rather than be reported a spurious success.
	if tmpl.Resource.Action == "delete" || (tmpl.Resource.SuccessCondition == "" && tmpl.Resource.FailureCondition == "") {
		rae.reportSucceeded(ctx, nodeID, tmpl, obj)
		return
	}
	// The resource exists and the agent is about to watch it (possibly for a long time); surface
	// that as Running so the node doesn't sit Pending and then jump straight to Succeeded.
	rae.reportRunning(nodeID)
	// Poll by name rather than watch when either the action is get — its live object never carries the
	// agent-resource label the informer selects on, so watch events would never arrive — or the parsed
	// conditions are empty (whitespace-only): there is nothing for the watch to match, but the resource
	// must still be read back once (as WaitResource does) so a mid-create deletion surfaces as a failure
	// instead of a spurious success. A single poll of zero requirements succeeds on the first read.
	if tmpl.Resource.Action == "get" || (len(rc.successReqs) == 0 && len(rc.failReqs) == 0) {
		go rae.pollResource(ctx, nodeID, tmpl, gvr, namespaced, obj, rc)
		return
	}
	if err = rae.ensureInformer(ctx, gvr, obj.GetNamespace()); err != nil {
		logger.WithError(err).Error(ctx, "failed to watch resource")
		rae.report(nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: fmt.Sprintf("failed to watch resource: %v", err)})
		return
	}
	// Close the create-to-watch gap: the object was created before the informer synced, so a delete
	// in that window yields neither an initial-list nor a watch event and the node would wait for an
	// event that never comes. The watch is now established, so confirm the object still exists; if it
	// vanished, report it rather than watching forever. Mirrors WaitResource's deleted-mid-check.
	ns := obj.GetNamespace()
	if namespaced && ns == "" {
		ns = rae.Namespace
	}
	if _, getErr := rae.getResource(ctx, gvr, namespaced, ns, obj.GetName()); apierrors.IsNotFound(getErr) {
		rae.report(nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: "the resource was deleted while its status was still being checked"})
	}
}

// pollResource polls a resource by name until its success/failure conditions are met, reporting a
// terminal result. Used for the get action, whose live object never carries the agent-resource
// label the watch informer selects on. Mirrors WaitResource in the per-pod executor.
func (rae *ResourceAgentExecutor) pollResource(ctx context.Context, nodeID string, tmpl *wfv1.Template, gvr schema.GroupVersionResource, namespaced bool, obj *unstructured.Unstructured, rc *resourceConditions) {
	logger := logging.RequireLoggerFromContext(ctx)
	ns := obj.GetNamespace()
	if namespaced && ns == "" {
		ns = rae.Namespace
	}
	name := obj.GetName()
	interval := env.LookupEnvDurationOr(ctx, "RESOURCE_STATE_CHECK_INTERVAL", 5*time.Second)
	if interval <= 0 {
		// a non-positive interval makes PollUntilContextCancel busy-loop with no delay
		interval = 5 * time.Second
	}
	err := wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		live, getErr := rae.getResource(ctx, gvr, namespaced, ns, name)
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				return false, argoerrors.Errorf(argoerrors.CodeNotFound, "The resource has been deleted while its status was still being checked.")
			}
			if argoerr.IsTransientErr(ctx, getErr) {
				return false, nil
			}
			return false, getErr
		}
		jsonBytes, mErr := json.Marshal(live)
		if mErr != nil {
			return false, mErr
		}
		stillWaiting, cErr := matchConditions(ctx, jsonBytes, rc.successReqs, rc.failReqs)
		if cErr == nil {
			rae.reportSucceeded(ctx, nodeID, tmpl, live)
			return true, nil
		}
		if stillWaiting {
			return false, nil
		}
		return false, cErr
	})
	if err != nil && !rae.isReported(nodeID) {
		logger.WithError(err).Info(ctx, "resource poll ended with failure")
		rae.report(nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: err.Error()})
	}
}

// report records a node's result for the patchWorker to flush. It never blocks (unlike a bounded
// channel) so a slow patch cannot stall the task workers, and it dedups so a node is reported once.
func (rae *ResourceAgentExecutor) report(nodeID string, result wfv1.NodeResult) {
	if nodeID == "" {
		return
	}
	rae.resultsMutex.Lock()
	defer rae.resultsMutex.Unlock()
	if rae.reported[nodeID] {
		return
	}
	rae.reported[nodeID] = true
	rae.pending[nodeID] = result
}

// reportRunning marks a node Running while the agent watches its resource. Unlike report() it does
// not set the reported latch, so the terminal result still follows; it is called once per node and
// skips if a terminal result already won the race, so it cannot regress or churn.
func (rae *ResourceAgentExecutor) reportRunning(nodeID string) {
	if nodeID == "" {
		return
	}
	rae.resultsMutex.Lock()
	defer rae.resultsMutex.Unlock()
	if rae.reported[nodeID] {
		return
	}
	rae.pending[nodeID] = wfv1.NodeResult{Phase: wfv1.NodeRunning}
}

func (rae *ResourceAgentExecutor) isReported(nodeID string) bool {
	rae.resultsMutex.Lock()
	defer rae.resultsMutex.Unlock()
	return rae.reported[nodeID]
}

func (rae *ResourceAgentExecutor) reportSucceeded(ctx context.Context, nodeID string, tmpl *wfv1.Template, obj *unstructured.Unstructured) {
	outputs, err := saveResourceParameters(ctx, tmpl, obj)
	if err != nil {
		rae.report(nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: fmt.Sprintf("failed to resolve output parameters: %v", err)})
		return
	}
	rae.report(nodeID, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: outputs})
}

func (rae *ResourceAgentExecutor) templateForNode(nodeID string) *wfv1.Template {
	rae.tasksMutex.RLock()
	defer rae.tasksMutex.RUnlock()
	return rae.tasks[nodeID]
}

func saveResourceParameters(ctx context.Context, tmpl *wfv1.Template, obj *unstructured.Unstructured) (*wfv1.Outputs, error) {
	if tmpl == nil || len(tmpl.Outputs.Parameters) == 0 {
		return nil, nil
	}
	action := ""
	if tmpl.Resource != nil {
		action = tmpl.Resource.Action
	}
	var jsonBytes []byte
	outputs := tmpl.Outputs.DeepCopy()
	for i, param := range outputs.Parameters {
		if param.ValueFrom == nil {
			continue
		}
		// A delete produces no queryable object; fall back to the configured default, as the
		// per-pod executor does when it has no resource name.
		if action == "delete" {
			output := ""
			if param.ValueFrom.Default != nil {
				output = param.ValueFrom.Default.String()
			}
			outputs.Parameters[i].Value = wfv1.AnyStringPtr(output)
			continue
		}
		var value string
		switch {
		case param.ValueFrom.JSONPath != "":
			// RelaxedJSONPathExpression is what `kubectl get -o jsonpath=` accepts
			expr, err := kubectlget.RelaxedJSONPathExpression(param.ValueFrom.JSONPath)
			if err != nil {
				return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "output parameter %q jsonPath failed to parse: %v", param.Name, err)
			}
			jp := jsonpath.New(param.Name)
			// match `kubectl get -o jsonpath`: an absent field yields empty, not an error
			jp.AllowMissingKeys(true)
			if err := jp.Parse(expr); err != nil {
				return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "output parameter %q jsonPath failed to parse: %v", param.Name, err)
			}
			var buf bytes.Buffer
			if err := jp.Execute(&buf, obj.Object); err != nil {
				return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "output parameter %q jsonPath evaluation failed: %v", param.Name, err)
			}
			value = buf.String()
		case param.ValueFrom.JQFilter != "":
			if jsonBytes == nil {
				var err error
				if jsonBytes, err = json.Marshal(obj); err != nil {
					return nil, err
				}
			}
			var err error
			if value, err = jqFilter(ctx, jsonBytes, param.ValueFrom.JQFilter); err != nil {
				return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "output parameter %q jqFilter evaluation failed: %v", param.Name, err)
			}
		default:
			continue
		}
		outputs.Parameters[i].Value = wfv1.AnyStringPtr(value)
	}
	return outputs, nil
}

func parseConditions(successCondition, failureCondition string) (*resourceConditions, error) {
	rc := &resourceConditions{}
	if successCondition != "" {
		successSelector, err := labels.Parse(successCondition)
		if err != nil {
			return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "success condition '%s' failed to parse: %v", successCondition, err)
		}
		rc.successReqs, _ = successSelector.Requirements()
	}
	if failureCondition != "" {
		failSelector, err := labels.Parse(failureCondition)
		if err != nil {
			return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "fail condition '%s' failed to parse: %v", failureCondition, err)
		}
		rc.failReqs, _ = failSelector.Requirements()
	}
	return rc, nil
}

// resolveGVR maps a kind to its real resource (plural) and scope via API discovery, so the informer
// watches the correct GVR even for CRDs whose plural is not a naive pluralization of the kind.
func (rae *ResourceAgentExecutor) resolveGVR(gvk schema.GroupVersionKind) (schema.GroupVersionResource, bool, error) {
	mapping, err := rae.RESTMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		// a freshly-created CRD may not be in cached discovery yet; refresh once and retry
		rae.RESTMapper.Reset()
		mapping, err = rae.RESTMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return schema.GroupVersionResource{}, false, fmt.Errorf("failed to resolve resource for %s: %w", gvk, err)
		}
	}
	return mapping.Resource, mapping.Scope.Name() == meta.RESTScopeNameNamespace, nil
}

// informerResync reads the agent's informer resync period, guarding against a negative value that
// would make the reflector resync in a tight LIST loop. Zero is valid and disables periodic resync.
func informerResync(ctx context.Context) time.Duration {
	resync := env.LookupEnvDurationOr(ctx, common.EnvAgentResourceInformerResync, 5*time.Minute)
	if resync < 0 {
		return 5 * time.Minute
	}
	return resync
}

func (rae *ResourceAgentExecutor) ensureInformer(ctx context.Context, gvr schema.GroupVersionResource, namespace string) error {
	key := informerKey{gvr: gvr, namespace: namespace}
	rae.informersMutex.Lock()
	existing, exists := rae.informers[key]
	rae.informersMutex.Unlock()
	if exists {
		return rae.waitInformerSynced(ctx, existing)
	}

	// Confirm the agent can actually list AND watch this GVR before relying on the informer. An
	// informer whose list/watch the agent service account may not perform never delivers an event
	// and never reports the failure to us — client-go's reflector just retries in the background —
	// so the node would hang until the workflow times out. One-shot List+Watch probes surface the
	// RBAC (or other) error synchronously, the way the per-pod executor's GET does. Probed without
	// the informers lock held so a slow API server can't stall event processing for other resources.
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	lister := rae.DynamicClient.Resource(gvr)
	nsLister := dynamic.ResourceInterface(lister)
	if namespace != "" {
		nsLister = lister.Namespace(namespace)
	}
	if _, probeErr := nsLister.List(probeCtx, metav1.ListOptions{Limit: 1}); probeErr != nil {
		return fmt.Errorf("cannot list %s: %w", gvr.Resource, probeErr)
	}
	// The informer needs watch, not just list; a role granting list but not watch would pass the
	// List probe and then hang silently as the reflector retries the denied watch forever.
	w, probeErr := nsLister.Watch(probeCtx, metav1.ListOptions{Limit: 1})
	if probeErr != nil {
		return fmt.Errorf("cannot watch %s: %w", gvr.Resource, probeErr)
	}
	w.Stop()

	rae.informersMutex.Lock()
	if inf, ok := rae.informers[key]; ok {
		rae.informersMutex.Unlock()
		return rae.waitInformerSynced(ctx, inf) // another worker created it while we probed
	}

	informer := dynamicinformer.NewFilteredDynamicInformer(rae.DynamicClient, gvr, namespace, informerResync(ctx),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		func(opts *metav1.ListOptions) {
			opts.LabelSelector = common.LabelKeyAgentResource + "=" + rae.WorkflowUID
		}).Informer()

	enqueue := func(o any, deleted bool) {
		if tombstone, ok := o.(cache.DeletedFinalStateUnknown); ok {
			o = tombstone.Obj
		}
		obj, ok := o.(*unstructured.Unstructured)
		if !ok {
			return
		}
		rae.eventQueue.Add(resourceEvent{
			obj:     obj,
			deleted: deleted,
			nodeID:  obj.GetAnnotations()[common.AnnotationKeyNodeID],
		})
	}
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(o any) { enqueue(o, false) },
		UpdateFunc: func(_, newObj any) { enqueue(newObj, false) },
		DeleteFunc: func(o any) { enqueue(o, true) },
	})
	if err != nil {
		rae.informersMutex.Unlock()
		return fmt.Errorf("failed to add resource event handler: %w", err)
	}
	go informer.Run(ctx.Done())
	rae.informers[key] = informer
	rae.informersMutex.Unlock()
	// Block until the initial list/watch is established so the caller's follow-up GET can't race
	// ahead of the watch (see createAndWatchResource).
	return rae.waitInformerSynced(ctx, informer)
}

// waitInformerSynced blocks until the informer's cache completes its initial sync, bounded by a
// timeout so a wedged informer fails the node loudly instead of hanging it forever.
func (rae *ResourceAgentExecutor) waitInformerSynced(ctx context.Context, informer cache.SharedIndexInformer) error {
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !cache.WaitForCacheSync(syncCtx.Done(), informer.HasSynced) {
		return fmt.Errorf("timed out waiting for resource informer cache to sync")
	}
	return nil
}

func (rae *ResourceAgentExecutor) runEventWorker(ctx context.Context) {
	for {
		ev, shutdown := rae.eventQueue.Get()
		if shutdown {
			return
		}
		rae.processEvent(ctx, ev)
		rae.eventQueue.Done(ev)
	}
}

func (rae *ResourceAgentExecutor) processEvent(ctx context.Context, ev resourceEvent) {
	obj := ev.obj
	if ev.nodeID == "" || obj == nil || rae.isReported(ev.nodeID) {
		return
	}
	// Only report for a node this agent is currently tracking. A nil template means the node is not
	// in our taskset — e.g. a completed-and-pruned node whose created object still exists and gets
	// re-listed after an agent restart. Reporting it would carry nil outputs (no template to read
	// output params from) and, since the controller applies taskset results unconditionally, would
	// clobber the node's already-recorded outputs. Skip it.
	tmpl := rae.templateForNode(ev.nodeID)
	if tmpl == nil {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"namespace": obj.GetNamespace(), "name": obj.GetName()})
	annotations := obj.GetAnnotations()
	rc, err := parseConditions(annotations[common.AnnotationKeySuccessCondition], annotations[common.AnnotationKeyFailureCondition])
	if err != nil {
		logger.WithError(err).Warn(ctx, "skipping resource with unparseable condition annotations")
		return
	}
	// Evaluate against the object carried on the event, not a fresh store read. On a delete event
	// obj is the last-known state, so a resource that met its success condition before being
	// deleted still reports Succeeded here; only a deletion that beat the condition falls through
	// to the failure below.
	if len(rc.successReqs) > 0 || len(rc.failReqs) > 0 {
		jsonBytes, mErr := json.Marshal(obj)
		if mErr != nil {
			logger.WithError(mErr).Error(ctx, "failed to marshal resource")
			return
		}
		stillWaiting, cErr := matchConditions(ctx, jsonBytes, rc.successReqs, rc.failReqs)
		switch {
		case cErr == nil:
			logger.Info(ctx, "resource met success conditions")
			rae.reportSucceeded(ctx, ev.nodeID, tmpl, obj)
			return
		case !stillWaiting:
			logger.WithError(cErr).Info(ctx, "resource met failure conditions")
			rae.report(ev.nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: cErr.Error()})
			return
		}
		// still waiting: a deletion (below) is terminal; otherwise the next event re-evaluates
	}
	if ev.deleted {
		logger.Info(ctx, "watched resource deleted before its conditions were met")
		rae.report(ev.nodeID, wfv1.NodeResult{
			Phase:   wfv1.NodeFailed,
			Message: "The resource has been deleted while its status was still being checked.",
		})
	}
}

func (rae *ResourceAgentExecutor) resolveManifest(ctx context.Context, tmpl *wfv1.Template) ([]byte, error) {
	if tmpl.Resource.ManifestFrom == nil || tmpl.Resource.ManifestFrom.Artifact == nil {
		return []byte(tmpl.Resource.Manifest), nil
	}
	name := tmpl.Resource.ManifestFrom.Artifact.Name
	art := tmpl.Inputs.GetArtifactByName(name)
	if art == nil {
		return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "manifestFrom artifact %q not found in template inputs", name)
	}
	driverArt := art.DeepCopy()
	if err := driverArt.Relocate(tmpl.ArchiveLocation); err != nil {
		return nil, err
	}
	if !driverArt.HasLocationOrKey() {
		return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "manifestFrom artifact %q has no location", name)
	}
	driver, err := artifacts.NewDriver(ctx, driverArt, rae)
	if err != nil {
		return nil, err
	}
	tmpDir, err := os.MkdirTemp("", "manifest-from-")
	if err != nil {
		return nil, argoerrors.InternalWrapError(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	downloadPath := filepath.Join(tmpDir, "download")
	err = driver.Load(ctx, driverArt, downloadPath)
	if err != nil {
		return nil, fmt.Errorf("manifestFrom artifact %q failed to load: %w", name, err)
	}

	manifestPath := filepath.Join(tmpDir, "manifest")
	isTar, isZip, err := detectArchiveType(ctx, *art, downloadPath)
	if err != nil {
		return nil, err
	}
	if err := extractArchive(ctx, isTar, isZip, downloadPath, manifestPath); err != nil {
		return nil, err
	}
	return os.ReadFile(manifestPath)
}

// GetSecret is used to fetch secrets for artifact drivers
func (rae *ResourceAgentExecutor) GetSecret(ctx context.Context, name, key string) (string, error) {
	secret, err := rae.ClientSet.CoreV1().Secrets(rae.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	val, ok := secret.Data[key]
	if !ok {
		return "", argoerrors.Errorf(argoerrors.CodeNotFound, "secret %q has no key %q", name, key)
	}
	return string(val), nil
}

// GetConfigMapKey is used to fetch config details
func (rae *ResourceAgentExecutor) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	cm, err := rae.ClientSet.CoreV1().ConfigMaps(rae.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	val, ok := cm.Data[key]
	if !ok {
		return "", argoerrors.Errorf(argoerrors.CodeNotFound, "configmap %q has no key %q", name, key)
	}
	return val, nil
}

// createResource runs the template's kubectl action against the manifest and returns the object to
// watch plus its resolved GVR. It is idempotent across agent restarts: if the resource already
// exists (a create re-run after a restart), it fetches and returns the existing object.
func (rae *ResourceAgentExecutor) createResource(ctx context.Context, resource *wfv1.ResourceTemplate, manifest []byte, nodeID string) (*unstructured.Unstructured, schema.GroupVersionResource, bool, error) {
	if len(manifest) == 0 {
		return nil, schema.GroupVersionResource{}, false, argoerrors.New(argoerrors.CodeBadRequest, "resource template has no manifest")
	}
	labeled, obj, err := withAgentMetadata(manifest, rae.WorkflowName, rae.WorkflowUID, nodeID, resource)
	if err != nil {
		return nil, schema.GroupVersionResource{}, false, err
	}
	// A manifest may name the object directly (metadata.name) or ask the server to assign one
	// (metadata.generateName); the assigned name is read back from kubectl's JSON output below.
	if obj.GetKind() == "" || (obj.GetName() == "" && obj.GetGenerateName() == "") {
		return nil, schema.GroupVersionResource{}, false, argoerrors.New(argoerrors.CodeBadRequest, "Kind and name (or generateName) are required but missing from the manifest")
	}
	gvr, namespaced, err := rae.resolveGVR(obj.GroupVersionKind())
	if err != nil {
		return nil, schema.GroupVersionResource{}, false, err
	}

	// Restart-safe: if this agent already created this node's resource, adopt it instead of creating
	// a duplicate. Matched by the workflow-UID label and node-ID annotation the agent stamps on every
	// object it creates. This is the only recovery path for generateName resources (a re-run create
	// gets a fresh server-assigned name, so it never collides), and it refuses foreign same-named
	// objects, which carry neither marker, when create hits AlreadyExists below.
	if existing, found, findErr := rae.findExistingResource(ctx, gvr, namespaced, obj, nodeID); findErr == nil && found {
		return existing, gvr, namespaced, nil
	}

	action := resource.Action
	if action == "" {
		action = "create"
	}

	// A plain create needs none of kubectl's semantics (flag passthrough, merge strategies,
	// server-side apply), so it goes through the DynamicClient directly and stays concurrent
	// across task workers — the create-heavy fan-outs the agent targets never queue on the
	// kubectl lock. findExistingResource above already adopted any object this agent created,
	// so an AlreadyExists from either path is a same-named object the agent does not own;
	// adopting it would make the agent monitor (and garbage-collect) an unrelated resource,
	// so fail instead.
	if action == "create" && len(resource.Flags) == 0 {
		created, createErr := rae.dynamicCreate(ctx, gvr, namespaced, obj)
		if createErr != nil {
			return nil, gvr, namespaced, createErr
		}
		return created, gvr, namespaced, nil
	}

	tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
	if err != nil {
		return nil, gvr, namespaced, argoerrors.InternalWrapError(err)
	}
	manifestPath := tmpFile.Name()
	defer func() { _ = os.Remove(manifestPath) }()
	if _, err = tmpFile.Write(labeled); err != nil {
		_ = tmpFile.Close()
		return nil, gvr, namespaced, argoerrors.InternalWrapError(err)
	}
	if err = tmpFile.Close(); err != nil {
		return nil, gvr, namespaced, argoerrors.InternalWrapError(err)
	}

	args, err := getKubectlArguments(action, manifestPath, resource.Flags, resource.MergeStrategy)
	if err != nil {
		return nil, gvr, namespaced, err
	}

	// The remaining actions (apply/patch/replace/delete/get, or any explicit flags) keep
	// in-process kubectl for behavior parity with the per-pod executor. kubectl binds flags
	// to package globals, so runKubectl serializes itself internally — capping these actions
	// at one in-flight kubectl regardless of ARGO_AGENT_TASK_WORKERS, a conscious trade-off
	// for that parity. The watch/report paths stay concurrent; only this step is serialized.
	out, err := runKubectlWithRetry(ctx, args...)
	if err != nil {
		return nil, gvr, namespaced, err
	}

	// create/apply/get/replace/patch print the object as JSON; delete prints a name. Fall back to
	// the manifest object when the output isn't a usable object (e.g. a delete).
	created := &unstructured.Unstructured{}
	if jsonErr := json.Unmarshal(out, created); jsonErr != nil || created.GetName() == "" {
		return obj, gvr, namespaced, nil
	}
	return created, gvr, namespaced, nil
}

// dynamicCreate POSTs the labeled manifest object through the DynamicClient, retrying
// transient errors with the same backoff runKubectlWithRetry uses. Unlike kubectl it
// holds no process-global state, so creates from concurrent task workers do not
// serialize. The server's response carries the assigned name for generateName manifests.
func (rae *ResourceAgentExecutor) dynamicCreate(ctx context.Context, gvr schema.GroupVersionResource, namespaced bool, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	ri := rae.DynamicClient.Resource(gvr)
	var created *unstructured.Unstructured
	err := retry.OnError(retry.DefaultBackoff, func(err error) bool {
		return argoerr.IsTransientErr(ctx, err)
	}, func() error {
		var createErr error
		if namespaced {
			ns := obj.GetNamespace()
			if ns == "" {
				ns = rae.Namespace
			}
			created, createErr = ri.Namespace(ns).Create(ctx, obj, metav1.CreateOptions{})
		} else {
			created, createErr = ri.Create(ctx, obj, metav1.CreateOptions{})
		}
		return createErr
	})
	return created, err
}

func (rae *ResourceAgentExecutor) getResource(ctx context.Context, gvr schema.GroupVersionResource, namespaced bool, namespace, name string) (*unstructured.Unstructured, error) {
	if namespaced {
		return rae.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	return rae.DynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
}

// findExistingResource looks for a resource this agent already created for the given node, matched by
// the workflow-UID label and node-ID annotation it stamps on every created object. It makes create
// idempotent across agent restarts, including for generateName resources whose server-assigned name
// a re-run create could never recover by name. The LIST hits the API server directly (strongly
// consistent), so a resource this agent created is always found.
func (rae *ResourceAgentExecutor) findExistingResource(ctx context.Context, gvr schema.GroupVersionResource, namespaced bool, obj *unstructured.Unstructured, nodeID string) (*unstructured.Unstructured, bool, error) {
	opts := metav1.ListOptions{LabelSelector: common.LabelKeyAgentResource + "=" + rae.WorkflowUID}
	var list *unstructured.UnstructuredList
	var err error
	if namespaced {
		ns := obj.GetNamespace()
		if ns == "" {
			ns = rae.Namespace
		}
		list, err = rae.DynamicClient.Resource(gvr).Namespace(ns).List(ctx, opts)
	} else {
		list, err = rae.DynamicClient.Resource(gvr).List(ctx, opts)
	}
	if err != nil {
		return nil, false, err
	}
	for i := range list.Items {
		if list.Items[i].GetAnnotations()[common.AnnotationKeyNodeID] == nodeID {
			return &list.Items[i], true, nil
		}
	}
	return nil, false, nil
}

func (rae *ResourceAgentExecutor) runResourceAgentWorker(ctx context.Context, taskQueue workqueue.TypedInterface[string]) {
	for {
		nodeID, shutdown := taskQueue.Get()
		if shutdown {
			return
		}
		rae.createAndWatchResource(ctx, rae.templateForNode(nodeID), nodeID)
		taskQueue.Done(nodeID)
	}
}

func (rae *ResourceAgentExecutor) patchWorker(ctx context.Context, taskSetInterface v1alpha1.WorkflowTaskSetInterface, requeueTime time.Duration) {
	ticker := time.NewTicker(requeueTime)
	defer ticker.Stop()
	logger := logging.RequireLoggerFromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rae.resultsMutex.Lock()
			if len(rae.pending) == 0 {
				rae.resultsMutex.Unlock()
				continue
			}
			batch := rae.pending
			rae.pending = map[string]wfv1.NodeResult{}
			rae.resultsMutex.Unlock()

			if err := patchTaskSetStatusNodes(ctx, taskSetInterface, rae.WorkflowName, batch); err != nil {
				if !argoerr.IsTransientErr(ctx, err) {
					// Non-transient (RBAC, validation, deleted taskset): retrying the same payload
					// forever would hang the workflow. Propagate the failure to the nodes as
					// NodeError so they terminate, matching AgentExecutor.patchWorker.
					logger.WithError(err).Error(ctx, "TaskSet Patch Failed; marking nodes errored to avoid a hang")
					errored := make(map[string]wfv1.NodeResult, len(batch))
					for nodeID := range batch {
						errored[nodeID] = wfv1.NodeResult{
							Phase:   wfv1.NodeError,
							Message: fmt.Sprintf("resource monitored but an error occurred when patching its result: %s", err),
						}
					}
					rae.requeueResults(errored)
					continue
				}
				// Transient: keep the real results and retry next tick rather than clobbering them.
				logger.WithError(err).Error(ctx, "TaskSet Patch Failed; will retry")
				rae.requeueResults(batch)
				continue
			}
			logger.Info(ctx, "Patched TaskSet")
		}
	}
}

// requeueResults returns un-patched results to pending so the next tick retries them, without
// overwriting any newer result that arrived for the same node in the meantime.
func (rae *ResourceAgentExecutor) requeueResults(batch map[string]wfv1.NodeResult) {
	rae.resultsMutex.Lock()
	defer rae.resultsMutex.Unlock()
	for nodeID, result := range batch {
		if _, ok := rae.pending[nodeID]; !ok {
			rae.pending[nodeID] = result
		}
	}
}

// Agent runs the resource agent
func (rae *ResourceAgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrashWithContext(ctx, runtimeutil.PanicHandlers...)
	taskWorkers := env.LookupEnvIntOr(ctx, common.EnvAgentTaskWorkers, 16)
	requeueTime := env.LookupEnvDurationOr(ctx, common.EnvAgentPatchRate, 10*time.Second)

	logger := logging.RequireLoggerFromContext(ctx)
	// time.NewTicker panics on a non-positive interval; a bad ARGO_AGENT_PATCH_RATE must not crash-loop the agent.
	if requeueTime <= 0 {
		logger.WithField("requeueTime", requeueTime).Warn(ctx, "invalid ARGO_AGENT_PATCH_RATE; falling back to 10s")
		requeueTime = 10 * time.Second
	}
	// `for range taskWorkers` starts zero consumers when this is non-positive, so every task would
	// sit on the queue forever. Clamp to the default, as requeueTime does above.
	if taskWorkers <= 0 {
		logger.WithField("taskWorkers", taskWorkers).Warn(ctx, "invalid ARGO_AGENT_TASK_WORKERS; falling back to 16")
		taskWorkers = 16
	}
	logger.WithField("taskWorkers", taskWorkers).
		WithField("requeueTime", requeueTime).
		Info(ctx, "Starting Agent")

	// Derived so the informer handlers can stop the agent early (the self-exits below).
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// A workqueue (like eventQueue) never blocks Add, so the informer's delivery goroutine can't
	// stall when a large fan-out of tasks outpaces the workers — unlike a bounded channel send.
	taskQueue := workqueue.NewTyped[string]()

	resync := informerResync(ctx)
	factory := externalversions.NewSharedInformerFactoryWithOptions(rae.WorkflowInterface, resync,
		externalversions.WithNamespace(rae.Namespace),
		// Only this workflow's own taskset. The taskset is named after the workflow and is
		// created solely by the controller, so a name field selector is spoof-proof: a label
		// (mutable, user-settable) is not — anyone able to create a taskset could otherwise stamp
		// our UID on it and have us create resources for their tasks under the agent's SA. This
		// mirrors the HTTP/plugin AgentExecutor's metadata.name selector.
		externalversions.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.FieldSelector = "metadata.name=" + rae.WorkflowName
		}))
	informer := factory.Argoproj().V1alpha1().WorkflowTaskSets().Informer()

	// enqueue tracks tasks by node ID, adding only ones not seen before (safe for AddFunc re-lists
	// on restart and UpdateFunc), and enqueues the new node IDs without holding the lock.
	enqueue := func(ts *wfv1.WorkflowTaskSet) {
		if ts == nil {
			return
		}
		// Defense in depth behind the name field selector: never ingest tasks from a taskset that
		// is not our workflow's own, so a spoofed taskset can never make us create its resources.
		if ts.Name != rae.WorkflowName {
			return
		}
		rae.tasksMutex.Lock()
		var newIDs []string
		for nodeID, template := range ts.Spec.Tasks {
			if _, seen := rae.tasks[nodeID]; seen {
				continue
			}
			rae.tasks[nodeID] = &template
			newIDs = append(newIDs, nodeID)
		}
		rae.tasksMutex.Unlock()
		for _, nodeID := range newIDs {
			taskQueue.Add(nodeID)
		}
	}

	// The controller deletes the agent pod when the workflow completes; these self-exits are
	// belt-and-suspenders (mirroring AgentExecutor) so a missed teardown can't leave the agent
	// running forever — stop when our taskset is marked completed or is deleted.
	upsert := func(ts *wfv1.WorkflowTaskSet) {
		if ts != nil && IsWorkflowCompleted(ts) {
			logger.Info(ctx, "Workflow completed; stopping resource agent")
			cancel()
			return
		}
		enqueue(ts)
	}

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			ts := asTaskSet(obj)
			logger.WithField("name", tsName(ts)).Info(ctx, "WorkflowTaskSet added")
			upsert(ts)
		},
		UpdateFunc: func(_, obj any) {
			ts := asTaskSet(obj)
			logger.WithField("name", tsName(ts)).Info(ctx, "WorkflowTaskSet updated")
			upsert(ts)
		},
		DeleteFunc: func(obj any) {
			logger.WithField("name", tsName(asTaskSet(obj))).Info(ctx, "WorkflowTaskSet deleted; stopping resource agent")
			cancel()
		},
	})
	if err != nil {
		return err
	}

	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return fmt.Errorf("failed to sync WorkflowTaskSet informer cache")
	}

	for range taskWorkers {
		go rae.runResourceAgentWorker(ctx, taskQueue)
	}

	go rae.patchWorker(ctx, rae.WorkflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(rae.Namespace), requeueTime)
	go rae.runEventWorker(ctx)
	go func() {
		<-ctx.Done()
		taskQueue.ShutDown()
		rae.eventQueue.ShutDown()
	}()

	<-ctx.Done()
	return nil
}

// withAgentMetadata parses a single-object manifest and re-marshals it with the UID-valued
// agent-resource label (informer selection) plus the success/failure conditions and owning
// node ID as annotations, making every event self-describing — handlers can evaluate and
// attribute results with no agent-side state, and a restarted agent's re-list resumes cleanly.
// It returns both the re-marshaled manifest and the parsed object.
// Note: single-document manifests only; multi-doc is rejected rather than silently partly
// applied. A future enhancement could split on YAML doc boundaries and label/create/watch each.
func withAgentMetadata(manifest []byte, workflowName, workflowUID, nodeID string, resource *wfv1.ResourceTemplate) ([]byte, *unstructured.Unstructured, error) {
	if common.ManifestDocCount(manifest) > 1 {
		return nil, nil, argoerrors.New(argoerrors.CodeBadRequest, "agent-based resource templates support only a single manifest document")
	}
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(manifest, &obj.Object); err != nil {
		return nil, nil, argoerrors.New(argoerrors.CodeBadRequest, err.Error())
	}
	// Inject the workflow ownerReference so the created object is garbage-collected with the workflow.
	// Done here (not in the controller, as executeResource does) because the controller never sees a
	// manifestFrom manifest's content, so injecting there would silently skip manifestFrom objects.
	if resource.SetOwnerReference {
		owner := &metav1.ObjectMeta{Name: workflowName, UID: types.UID(workflowUID)}
		ref := *metav1.NewControllerRef(owner, wfv1.SchemeGroupVersion.WithKind(apiworkflow.WorkflowKind))
		obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ref))
	}
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[common.LabelKeyAgentResource] = workflowUID
	obj.SetLabels(labels)
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[common.AnnotationKeyNodeID] = nodeID
	if resource.SuccessCondition != "" {
		annotations[common.AnnotationKeySuccessCondition] = resource.SuccessCondition
	}
	if resource.FailureCondition != "" {
		annotations[common.AnnotationKeyFailureCondition] = resource.FailureCondition
	}
	obj.SetAnnotations(annotations)
	out, err := yaml.Marshal(obj.Object)
	if err != nil {
		return nil, nil, err
	}
	return out, obj, nil
}

// asTaskSet extracts a *WorkflowTaskSet from an informer event object, unwrapping
// the delete tombstone that DeleteFunc may receive.
func asTaskSet(obj any) *wfv1.WorkflowTaskSet {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	ts, _ := obj.(*wfv1.WorkflowTaskSet)
	return ts
}

func tsName(ts *wfv1.WorkflowTaskSet) string {
	if ts == nil {
		return ""
	}
	return ts.Name
}
