package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	RESTClient        rest.Interface
	RESTMapper        meta.ResettableRESTMapper
	Namespace         string

	tasksMutex sync.RWMutex
	tasks      map[string]*wfv1.Template // keyed by node ID

	kubeCTLMutex sync.Mutex

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
func NewResourceAgentExecutor(clientSet kubernetes.Interface, restClient rest.Interface, config *rest.Config, namespace, workflowName, workflowUID string) *ResourceAgentExecutor {
	return &ResourceAgentExecutor{
		WorkflowName:      workflowName,
		WorkflowUID:       workflowUID,
		ClientSet:         clientSet,
		DynamicClient:     dynamic.NewForConfigOrDie(config),
		WorkflowInterface: workflow.NewForConfigOrDie(config),
		RESTClient:        restClient,
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

	// A delete leaves no live object to watch, and zero requirements (empty or whitespace-only
	// conditions) mean there is nothing to wait for: succeed immediately, as WaitResource does.
	if tmpl.Resource.Action == "delete" || (len(rc.successReqs) == 0 && len(rc.failReqs) == 0) {
		rae.reportSucceeded(ctx, nodeID, tmpl, obj)
		return
	}
	// A get does not write the agent-resource label onto the live object, so the label-filtered
	// informer would never deliver its events. Poll it by name instead, as WaitResource does.
	if tmpl.Resource.Action == "get" {
		go rae.pollResource(ctx, nodeID, tmpl, gvr, namespaced, obj, rc)
		return
	}
	if err = rae.ensureInformer(ctx, gvr, obj.GetNamespace()); err != nil {
		logger.WithError(err).Error(ctx, "failed to watch resource")
		rae.report(nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: fmt.Sprintf("failed to watch resource: %v", err)})
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
	_, exists := rae.informers[key]
	rae.informersMutex.Unlock()
	if exists {
		return nil
	}

	// Confirm the agent can actually watch this GVR before relying on the informer. An informer
	// whose list/watch the agent service account may not perform never delivers an event and never
	// reports the failure to us — client-go's reflector just retries in the background — so the
	// node would hang until the workflow times out. A one-shot List surfaces the RBAC (or other)
	// error synchronously, the way the per-pod executor's GET does. Probed without the informers
	// lock held so a slow API server can't stall event processing for other resources.
	// ponytail: probes list, not watch; a role granting list but not watch (rare) still hangs.
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	lister := rae.DynamicClient.Resource(gvr)
	var probeErr error
	if namespace != "" {
		_, probeErr = lister.Namespace(namespace).List(probeCtx, metav1.ListOptions{Limit: 1})
	} else {
		_, probeErr = lister.List(probeCtx, metav1.ListOptions{Limit: 1})
	}
	if probeErr != nil {
		return fmt.Errorf("cannot watch %s: %w", gvr.Resource, probeErr)
	}

	rae.informersMutex.Lock()
	defer rae.informersMutex.Unlock()
	if _, ok := rae.informers[key]; ok {
		return nil // another worker created it while we probed
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
		return fmt.Errorf("failed to add resource event handler: %w", err)
	}
	go informer.Run(ctx.Done())
	rae.informers[key] = informer
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
	isTar, isZip := false, false
	switch {
	case art.GetArchive().None != nil:
	case art.GetArchive().Tar != nil:
		isTar = true
	case art.GetArchive().Zip != nil:
		isZip = true
	default:
		if isTar, err = isTarball(ctx, downloadPath); err != nil {
			return nil, err
		}
	}
	switch {
	case isTar:
		err = untar(downloadPath, manifestPath)
	case isZip:
		err = unzip(ctx, downloadPath, manifestPath)
	default:
		err = os.Rename(downloadPath, manifestPath)
	}
	if err != nil {
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
	labeled, obj, err := withAgentMetadata(manifest, rae.WorkflowUID, nodeID, resource)
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

	action := resource.Action
	if action == "" {
		action = "create"
	}
	args, err := getKubectlArguments(action, manifestPath, resource.Flags, resource.MergeStrategy)
	if err != nil {
		return nil, gvr, namespaced, err
	}

	// runKubectl mutates global os.Args and kubectl's fatal handler; serialize across workers.
	rae.kubeCTLMutex.Lock()
	out, err := runKubectlWithRetry(ctx, args...)
	rae.kubeCTLMutex.Unlock()
	if err != nil {
		// restart-safe: a re-run create for a resource this agent already made returns the existing one
		if strings.Contains(err.Error(), "AlreadyExists") && obj.GetName() != "" {
			ns := obj.GetNamespace()
			if namespaced && ns == "" {
				ns = rae.Namespace
			}
			if existing, getErr := rae.getResource(ctx, gvr, namespaced, ns, obj.GetName()); getErr == nil {
				return existing, gvr, namespaced, nil
			}
		}
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

func (rae *ResourceAgentExecutor) getResource(ctx context.Context, gvr schema.GroupVersionResource, namespaced bool, namespace, name string) (*unstructured.Unstructured, error) {
	if namespaced {
		return rae.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	return rae.DynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
}

func (rae *ResourceAgentExecutor) runResourceAgentWorker(ctx context.Context, taskQueue chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case nodeID := <-taskQueue:
			rae.createAndWatchResource(ctx, rae.templateForNode(nodeID), nodeID)
		}
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

			patch, err := json.Marshal(map[string]any{"status": wfv1.WorkflowTaskSetStatus{Nodes: batch}})
			if err != nil {
				logger.WithError(err).Error(ctx, "Generating Patch Failed")
				rae.requeueResults(batch)
				continue
			}
			err = retry.OnError(wait.Backoff{
				Duration: time.Second,
				Factor:   2,
				Jitter:   0.1,
				Steps:    10,
				Cap:      60 * time.Second,
			}, func(retryErr error) bool {
				return argoerr.IsTransientErr(ctx, retryErr)
			}, func() error {
				_, patchErr := taskSetInterface.Patch(ctx, rae.WorkflowName, types.MergePatchType, patch, metav1.PatchOptions{}, "status")
				return patchErr
			})
			if err != nil {
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
	logger.WithField("taskWorkers", taskWorkers).
		WithField("requeueTime", requeueTime).
		Info(ctx, "Starting Agent")

	taskQueue := make(chan string, 32)

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
			taskQueue <- nodeID
		}
	}

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			ts := asTaskSet(obj)
			logger.WithField("name", tsName(ts)).Info(ctx, "WorkflowTaskSet added")
			enqueue(ts)
		},
		UpdateFunc: func(_, obj any) {
			ts := asTaskSet(obj)
			logger.WithField("name", tsName(ts)).Info(ctx, "WorkflowTaskSet updated")
			enqueue(ts)
		},
		DeleteFunc: func(obj any) {
			if ts := asTaskSet(obj); ts != nil {
				logger.WithField("name", ts.Name).Info(ctx, "WorkflowTaskSet deleted")
			}
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
// ponytail: single-document manifests only; multi-doc is rejected rather than silently partly
// applied. Upgrade path: split on YAML doc boundaries and label/create/watch each.
func withAgentMetadata(manifest []byte, workflowUID, nodeID string, resource *wfv1.ResourceTemplate) ([]byte, *unstructured.Unstructured, error) {
	if common.ManifestDocCount(manifest) > 1 {
		return nil, nil, argoerrors.New(argoerrors.CodeBadRequest, "agent-based resource templates support only a single manifest document")
	}
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(manifest, &obj.Object); err != nil {
		return nil, nil, argoerrors.New(argoerrors.CodeBadRequest, err.Error())
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
