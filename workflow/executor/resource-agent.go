package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	kubectlget "k8s.io/kubectl/pkg/cmd/get"
	"sigs.k8s.io/yaml"
)

// taskKey identifies a task within a WorkflowTaskSet; Spec.Tasks is keyed by node ID.
type taskKey struct {
	UID    string
	NodeID string
}

type informerKey struct {
	gvr       schema.GroupVersionResource
	namespace string
}

type resourceEvent struct {
	gvr       schema.GroupVersionResource
	namespace string
	name      string
	// nodeID is carried on the event (from the object's node-id annotation) so a deletion —
	// where the object is gone from the store — can still be attributed to its node.
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
	Namespace         string
	TasksMutex        *sync.RWMutex
	Tasks             map[taskKey]*wfv1.Template
	kubeCTLMutex      *sync.Mutex
	informersMutex    sync.Mutex
	informers         map[informerKey]cache.SharedIndexInformer
	eventQueue        workqueue.TypedInterface[resourceEvent]
	responseQueue     chan response
	reportedMutex     sync.Mutex
	reported          map[string]bool
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
		Namespace:         namespace,
		TasksMutex:        &sync.RWMutex{},
		Tasks:             map[taskKey]*wfv1.Template{},
		kubeCTLMutex:      &sync.Mutex{},
		eventQueue:        workqueue.NewTyped[resourceEvent](),
		responseQueue:     make(chan response, 32),
		reported:          map[string]bool{},
	}
}

func (rae *ResourceAgentExecutor) createAndWatchResource(ctx context.Context, tmpl *wfv1.Template, nodeID string) {
	if tmpl.Resource == nil {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx)
	// fail fast: don't create a resource whose conditions can never be evaluated
	// this is probably the ideal behaviour? It diverges from existing but presumably is fine
	if _, err := parseConditions(tmpl.Resource.SuccessCondition, tmpl.Resource.FailureCondition); err != nil {
		logger.WithError(err).Error(ctx, "invalid resource conditions")
		rae.report(ctx, nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: err.Error()})
		return
	}
	manifest, err := rae.resolveManifest(ctx, tmpl)
	if err != nil {
		logger.WithError(err).Error(ctx, "failed to resolve resource manifest")
		rae.report(ctx, nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: fmt.Sprintf("failed to resolve manifest: %v", err)})
		return
	}
	obj, err := rae.createResource(ctx, tmpl.Resource, manifest, nodeID)
	if err != nil {
		logger.WithError(err).Error(ctx, "failed to create resource")
		rae.report(ctx, nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: fmt.Sprintf("failed to create resource: %v", err)})
		return
	}
	logger.WithFields(logging.Fields{"namespace": obj.GetNamespace(), "name": obj.GetName(), "kind": obj.GetKind()}).Info(ctx, "Created resource")
	if tmpl.Resource.SuccessCondition == "" && tmpl.Resource.FailureCondition == "" {
		// nothing to wait for: succeed immediately, as WaitResource does
		rae.reportSucceeded(ctx, nodeID, tmpl, obj)
		return
	}
	rae.ensureInformerFor(ctx, obj)
}

func (rae *ResourceAgentExecutor) report(ctx context.Context, nodeID string, result wfv1.NodeResult) {
	if nodeID == "" {
		return
	}
	rae.reportedMutex.Lock()
	already := rae.reported[nodeID]
	rae.reported[nodeID] = true
	rae.reportedMutex.Unlock()
	if already {
		return
	}
	select {
	case rae.responseQueue <- response{NodeID: nodeID, Result: &result}:
	case <-ctx.Done():
	}
}

func (rae *ResourceAgentExecutor) isReported(nodeID string) bool {
	rae.reportedMutex.Lock()
	defer rae.reportedMutex.Unlock()
	return rae.reported[nodeID]
}

func (rae *ResourceAgentExecutor) reportSucceeded(ctx context.Context, nodeID string, tmpl *wfv1.Template, obj *unstructured.Unstructured) {
	outputs, err := saveResourceParameters(ctx, tmpl, obj)
	if err != nil {
		rae.report(ctx, nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: fmt.Sprintf("failed to resolve output parameters: %v", err)})
		return
	}
	rae.report(ctx, nodeID, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: outputs})
}

func (rae *ResourceAgentExecutor) templateForNode(nodeID string) *wfv1.Template {
	rae.TasksMutex.RLock()
	defer rae.TasksMutex.RUnlock()
	for k, tmpl := range rae.Tasks {
		if k.NodeID == nodeID {
			return tmpl
		}
	}
	return nil
}

func saveResourceParameters(ctx context.Context, tmpl *wfv1.Template, obj *unstructured.Unstructured) (*wfv1.Outputs, error) {
	if tmpl == nil || len(tmpl.Outputs.Parameters) == 0 {
		return nil, nil
	}
	var jsonBytes []byte
	outputs := tmpl.Outputs.DeepCopy()
	for i, param := range outputs.Parameters {
		if param.ValueFrom == nil {
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

func (rae *ResourceAgentExecutor) ensureInformerFor(ctx context.Context, obj *unstructured.Unstructured) cache.SharedIndexInformer {
	gvk := obj.GroupVersionKind()
	gvr := gvk.GroupVersion().WithResource(inferPluralName(gvk.Kind))
	return rae.ensureInformer(ctx, gvr, obj.GetNamespace())
}

func (rae *ResourceAgentExecutor) ensureInformer(ctx context.Context, gvr schema.GroupVersionResource, namespace string) cache.SharedIndexInformer {
	key := informerKey{gvr: gvr, namespace: namespace}
	rae.informersMutex.Lock()
	defer rae.informersMutex.Unlock()
	if informer, ok := rae.informers[key]; ok {
		return informer
	}

	resync := env.LookupEnvDurationOr(ctx, "ARGO_AGENT_RESOURCE_INFORMER_RESYNC", 10*time.Minute)
	informer := dynamicinformer.NewFilteredDynamicInformer(rae.DynamicClient, gvr, namespace, resync,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		func(opts *metav1.ListOptions) {
			opts.LabelSelector = common.LabelKeyAgentResource + "=" + rae.WorkflowUID
		}).Informer()

	enqueue := func(o any) {
		if tombstone, ok := o.(cache.DeletedFinalStateUnknown); ok {
			o = tombstone.Obj
		}
		obj, ok := o.(*unstructured.Unstructured)
		if !ok {
			return
		}
		rae.eventQueue.Add(resourceEvent{
			gvr:       gvr,
			namespace: obj.GetNamespace(),
			name:      obj.GetName(),
			nodeID:    obj.GetAnnotations()[common.AnnotationKeyNodeID],
		})
	}
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    enqueue,
		UpdateFunc: func(_, newObj any) { enqueue(newObj) },
		DeleteFunc: enqueue,
	})
	go informer.Run(ctx.Done())
	if rae.informers == nil {
		rae.informers = map[informerKey]cache.SharedIndexInformer{}
	}
	rae.informers[key] = informer
	return informer
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
	logger := logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"namespace": ev.namespace, "name": ev.name, "resource": ev.gvr.Resource})
	if ev.nodeID == "" || rae.isReported(ev.nodeID) {
		return
	}
	rae.informersMutex.Lock()
	informer := rae.informers[informerKey{gvr: ev.gvr, namespace: ev.namespace}]
	rae.informersMutex.Unlock()
	if informer == nil {
		return
	}
	item, exists, err := informer.GetStore().GetByKey(ev.namespace + "/" + ev.name)
	if err != nil {
		logger.WithError(err).Error(ctx, "failed to read resource from informer store")
		return
	}
	if !exists {
		logger.Info(ctx, "watched resource deleted")
		rae.report(ctx, ev.nodeID, wfv1.NodeResult{
			Phase:   wfv1.NodeFailed,
			Message: "The resource has been deleted while its status was still being checked.",
		})
		return
	}
	obj, ok := item.(*unstructured.Unstructured)
	if !ok {
		return
	}
	annotations := obj.GetAnnotations()
	rc, err := parseConditions(annotations[common.AnnotationKeySuccessCondition], annotations[common.AnnotationKeyFailureCondition])
	if err != nil {
		logger.WithError(err).Warn(ctx, "skipping resource with unparseable condition annotations")
		return
	}
	if len(rc.successReqs) == 0 && len(rc.failReqs) == 0 {
		return
	}
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		logger.WithError(err).Error(ctx, "failed to marshal resource")
		return
	}
	stillWaiting, err := matchConditions(ctx, jsonBytes, rc.successReqs, rc.failReqs)
	switch {
	case err == nil:
		logger.Info(ctx, "resource met success conditions")
		rae.reportSucceeded(ctx, ev.nodeID, rae.templateForNode(ev.nodeID), obj)
	case stillWaiting:
		// neither condition met yet; the next event re-enqueues this object
	default:
		logger.WithError(err).Info(ctx, "resource met failure conditions")
		rae.report(ctx, ev.nodeID, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: err.Error()})
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
	if err := driver.Load(ctx, driverArt, downloadPath); err != nil {
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

func (rae *ResourceAgentExecutor) createResource(ctx context.Context, resource *wfv1.ResourceTemplate, manifest []byte, nodeID string) (*unstructured.Unstructured, error) {
	if len(manifest) > 0 {
		labeled, err := withAgentMetadata(manifest, rae.WorkflowUID, nodeID, resource)
		if err != nil {
			return nil, err
		}
		manifest = labeled
	}

	tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
	if err != nil {
		return nil, argoerrors.InternalWrapError(err)
	}
	manifestPath := tmpFile.Name()
	defer func() { _ = os.Remove(manifestPath) }()
	if _, err := tmpFile.Write(manifest); err != nil {
		_ = tmpFile.Close()
		return nil, argoerrors.InternalWrapError(err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, argoerrors.InternalWrapError(err)
	}

	args, err := getKubectlArguments("create", manifestPath, resource.Flags, resource.MergeStrategy)
	if err != nil {
		return nil, err
	}

	var out []byte
	err = retry.OnError(retry.DefaultBackoff, func(err error) bool {
		return argoerr.IsTransientErr(ctx, err)
	}, func() error {
		// runKubectl mutates global os.Args and kubectl's fatal handler; serialize across workers.
		rae.kubeCTLMutex.Lock()
		defer rae.kubeCTLMutex.Unlock()
		var runErr error
		out, runErr = runKubectl(ctx, args...)
		return runErr
	})
	if err != nil {
		var exErr *exec.ExitError
		if errors.As(err, &exErr) {
			err = argoerrors.Wrap(err, argoerrors.CodeBadRequest, strings.TrimSpace(string(exErr.Stderr)))
		} else {
			err = argoerrors.Wrap(err, argoerrors.CodeBadRequest, err.Error())
		}
		return nil, argoerrors.Wrap(err, argoerrors.CodeBadRequest, "no more retries "+err.Error())
	}

	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal(out, obj); err != nil {
		return nil, err
	}
	if obj.GetName() == "" || obj.GetKind() == "" {
		return nil, argoerrors.New(argoerrors.CodeBadRequest, "Kind and name are both required but at least one of them is missing from the manifest")
	}
	return obj, nil
}

func (rae *ResourceAgentExecutor) runResourceAgentWorker(ctx context.Context, taskQueue chan taskKey) {
	for {
		select {
		case <-ctx.Done():
			return
		case key := <-taskQueue:
			rae.TasksMutex.RLock()
			template := rae.Tasks[key]
			rae.TasksMutex.RUnlock()
			rae.createAndWatchResource(ctx, template, key.NodeID)
		}
	}
}

func (rae *ResourceAgentExecutor) patchWorker(ctx context.Context, taskSetInterface v1alpha1.WorkflowTaskSetInterface, requeueTime time.Duration) {
	ticker := time.NewTicker(requeueTime)
	defer ticker.Stop()
	nodeResults := map[string]wfv1.NodeResult{}
	logger := logging.RequireLoggerFromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-rae.responseQueue:
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
				Steps:    10,
				Cap:      60 * time.Second,
			}, func(retryErr error) bool {
				return argoerr.IsTransientErr(ctx, retryErr)
			}, func() error {
				_, patchErr := taskSetInterface.Patch(ctx, rae.WorkflowName, types.MergePatchType, patch, metav1.PatchOptions{}, "status")
				return patchErr
			})
			if err != nil {
				if argoerr.IsTransientErr(ctx, err) {
					continue // keep the results; retry on the next tick
				}
				logger.WithError(err).Error(ctx, "TaskSet Patch Failed")
				// Non-transient: likely the patch contents. Propagate the error to the nodes
				// instead of retrying forever and deadlocking the workflow.
				for node := range nodeResults {
					nodeResults[node] = wfv1.NodeResult{
						Phase:   wfv1.NodeError,
						Message: fmt.Sprintf("resource processed successfully but an error occurred when patching its result: %s", err),
					}
				}
				continue
			}
			nodeResults = map[string]wfv1.NodeResult{}
			logger.Info(ctx, "Patched TaskSet")
		}
	}
}

// Agent runs the resource agent
func (rae *ResourceAgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrashWithContext(ctx, runtimeutil.PanicHandlers...)
	taskWorkers := env.LookupEnvIntOr(ctx, common.EnvAgentTaskWorkers, 16)
	requeueTime := env.LookupEnvDurationOr(ctx, common.EnvAgentPatchRate, 10*time.Second)

	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("taskWorkers", taskWorkers).
		WithField("requeueTime", requeueTime).
		Info(ctx, "Starting Agent")

	taskQueue := make(chan taskKey, 32)

	// requeueTime is the patch-flush rate, not a resync period; the informer gets its own.
	resync := env.LookupEnvDurationOr(ctx, "ARGO_AGENT_RESOURCE_INFORMER_RESYNC", 5*time.Minute)
	factory := externalversions.NewSharedInformerFactoryWithOptions(rae.WorkflowInterface, resync,
		externalversions.WithNamespace(rae.Namespace),
		// only this workflow's taskset — without this we would ingest (and create resources
		// for) every workflow's tasks in the namespace
		externalversions.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.LabelSelector = common.LabelKeyWorkflowUID + "=" + rae.WorkflowUID
		}))
	informer := factory.Argoproj().V1alpha1().WorkflowTaskSets().Informer()
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			ts := asTaskSet(obj)
			if ts == nil {
				return
			}
			logger.WithField("name", ts.Name).Info(ctx, "WorkflowTaskSet added")

			rae.TasksMutex.Lock()
			keys := make([]taskKey, 0, len(ts.Spec.Tasks))
			for nodeID, template := range ts.Spec.Tasks {
				k := taskKey{UID: string(ts.UID), NodeID: nodeID}
				rae.Tasks[k] = &template
				keys = append(keys, k)
			}
			rae.TasksMutex.Unlock()

			for _, k := range keys {
				taskQueue <- k // safe: not holding the lock
			}
		},
		UpdateFunc: func(_, obj any) {
			ts := asTaskSet(obj)
			if ts == nil {
				return
			}
			logger.WithField("name", ts.Name).Info(ctx, "WorkflowTaskSet updated")

			rae.TasksMutex.Lock()
			var newKeys []taskKey
			for nodeID, template := range ts.Spec.Tasks {
				k := taskKey{UID: string(ts.UID), NodeID: nodeID}
				if _, seen := rae.Tasks[k]; seen {
					continue // already tracked, nothing new to enqueue
				}
				rae.Tasks[k] = &template
				newKeys = append(newKeys, k)
			}
			rae.TasksMutex.Unlock()

			for _, k := range newKeys {
				taskQueue <- k // safe: not holding the lock
			}
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
// ponytail: single-document manifests only; multi-doc YAML would need splitting on "---".
func withAgentMetadata(manifest []byte, workflowUID, nodeID string, resource *wfv1.ResourceTemplate) ([]byte, error) {
	obj := unstructured.Unstructured{}
	if err := yaml.Unmarshal(manifest, &obj.Object); err != nil {
		return nil, argoerrors.New(argoerrors.CodeBadRequest, err.Error())
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
	return yaml.Marshal(obj.Object)
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
