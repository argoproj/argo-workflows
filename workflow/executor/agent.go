package executor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/client-go/util/retry"
	"k8s.io/gengo/namer"
	gengotypes "k8s.io/gengo/types"
	"sigs.k8s.io/yaml"

	goerrors "errors"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	executorplugins "github.com/argoproj/argo-workflows/v4/pkg/plugins/executor"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/env"
	"github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

type AgentExecutor struct {
	WorkflowName      string
	workflowUID       string
	ClientSet         kubernetes.Interface
	DynamicClient     dynamic.Interface
	WorkflowInterface workflow.Interface
	RESTClient        rest.Interface
	Namespace         string
	consideredTasks   *sync.Map
	plugins           []executorplugins.TemplateExecutor
	resourceInformer  *MonitoredResourceInformer

	pendingMu     sync.Mutex
	pendingTasks  map[string]pendingResourceTask
	responseQueue chan response
}

// pendingResourceTask holds the state handleDone needs to evaluate informer
// events for an in-flight resource template task.
type pendingResourceTask struct {
	successReqs labels.Requirements
	failReqs    labels.Requirements
	outputs     []wfv1.Parameter
}

type templateExecutor = func(ctx context.Context, nodeID string, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error)

func NewAgentExecutor(clientSet kubernetes.Interface, restClient rest.Interface, config *rest.Config, namespace, workflowName, workflowUID string, plugins []executorplugins.TemplateExecutor) *AgentExecutor {
	dynClient := dynamic.NewForConfigOrDie(config)
	ae := &AgentExecutor{
		ClientSet:         clientSet,
		RESTClient:        restClient,
		DynamicClient:     dynClient,
		Namespace:         namespace,
		WorkflowName:      workflowName,
		workflowUID:       workflowUID,
		WorkflowInterface: workflow.NewForConfigOrDie(config),
		consideredTasks:   &sync.Map{},
		plugins:           plugins,
		pendingTasks:      map[string]pendingResourceTask{},
		responseQueue:     make(chan response, 64),
	}
	ae.resourceInformer = NewMonitoredResourceInformer(dynClient, namespace, workflowName, 10*time.Minute, ae.handleDone)
	return ae
}

type task struct {
	NodeID   string
	Template wfv1.Template
}

type response struct {
	NodeID string
	Result *wfv1.NodeResult
}

func (ae *AgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrashWithContext(ctx, runtimeutil.PanicHandlers...)

	taskWorkers := env.LookupEnvIntOr(ctx, common.EnvAgentTaskWorkers, 16)
	requeueTime := env.LookupEnvDurationOr(ctx, common.EnvAgentPatchRate, 10*time.Second)
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("taskWorkers", taskWorkers).
		WithField("requeueTime", requeueTime).
		Info(ctx, "Starting Agent")

	taskQueue := make(chan task)
	taskSetInterface := ae.WorkflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(ae.Namespace)

	go ae.patchWorker(ctx, taskSetInterface, requeueTime)
	for range taskWorkers {
		go ae.taskWorker(ctx, taskQueue)
	}

	for {
		wfWatch, err := taskSetInterface.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + ae.WorkflowName})
		if err != nil {
			return err
		}

		for event := range wfWatch.ResultChan() {
			logger.WithField("event_type", event.Type).Info(ctx, "TaskSet Event")

			if event.Type == watch.Deleted {
				// We're done if the task set is deleted
				return nil
			}

			taskSet, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			if IsWorkflowCompleted(taskSet) {
				logger.Info(ctx, "Workflow completed... stopping agent")
				return nil
			}

			for nodeID, tmpl := range taskSet.Spec.Tasks {
				taskQueue <- task{NodeID: nodeID, Template: tmpl}
			}
		}
	}
}

func (ae *AgentExecutor) taskWorker(ctx context.Context, taskQueue chan task) {
	for {
		workTask, ok := <-taskQueue
		if !ok {
			break
		}
		nodeID, tmpl := workTask.NodeID, workTask.Template
		var logger logging.Logger
		ctx, logger = logging.RequireLoggerFromContext(ctx).WithField("nodeID", nodeID).InContext(ctx)

		// Do not work on tasks that have already been considered once, to prevent calling an endpoint more
		// than once unintentionally.
		if _, ok := ae.consideredTasks.LoadOrStore(nodeID, true); ok {
			logger.Info(ctx, "Task is already considered")
			continue
		}

		logger.Info(ctx, "Processing task")
		result, requeue, err := ae.processTask(ctx, nodeID, tmpl)
		if err != nil {
			logger.WithError(err).Error(ctx, "Error in agent task")
			result = &wfv1.NodeResult{
				Phase:   wfv1.NodeError,
				Message: fmt.Sprintf("error processing task: %s", err),
			}
			// Do not return or continue here, the "errored" result still needs to be propagated to the responseQueue below
		}

		logger.
			WithField("phase", result.Phase).
			WithField("message", result.Message).
			WithField("requeue", requeue).
			Info(ctx, "Sending result")

		if result.Phase != "" {
			ae.responseQueue <- response{NodeID: nodeID, Result: result}
		}
		if requeue > 0 {
			time.AfterFunc(requeue, func() {
				ae.consideredTasks.Delete(nodeID)

				taskQueue <- workTask
			})
		}
	}
}

func (ae *AgentExecutor) patchWorker(ctx context.Context, taskSetInterface v1alpha1.WorkflowTaskSetInterface, requeueTime time.Duration) {
	ticker := time.NewTicker(requeueTime)
	defer ticker.Stop()
	nodeResults := map[string]wfv1.NodeResult{}
	logger := logging.RequireLoggerFromContext(ctx)
	for {
		select {
		case res := <-ae.responseQueue:
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

			logger.Info(ctx, "Processing Patch")

			err = retry.OnError(wait.Backoff{
				Duration: time.Second,
				Factor:   2,
				Jitter:   0.1,
				Steps:    5,
				Cap:      30 * time.Second,
			}, func(retryErr error) bool {
				return errors.IsTransientErr(ctx, retryErr)
			}, func() error {
				_, patchErr := taskSetInterface.Patch(ctx, ae.WorkflowName, types.MergePatchType, patch, metav1.PatchOptions{}, "status")
				return patchErr
			})

			if err != nil && !errors.IsTransientErr(ctx, err) {
				logger.WithError(err).
					Error(ctx, "TaskSet Patch Failed")

				// If this is not a transient error, then it's likely that the contents of the patch have caused the error.
				// To avoid a deadlock with the workflow overall, or an infinite loop, fail and propagate the error messages
				// to the nodes.
				// If this is a transient error, then simply do nothing and another patch will be retried in the next tick.
				for node := range nodeResults {
					nodeResults[node] = wfv1.NodeResult{
						Phase:   wfv1.NodeError,
						Message: fmt.Sprintf("HTTP request completed successfully but an error occurred when patching its result: %s", err),
					}
				}
				continue
			}

			// Patch was successful, clear nodeResults for next iteration
			nodeResults = map[string]wfv1.NodeResult{}

			logger.Info(ctx, "Patched TaskSet")
		}
	}
}

func (ae *AgentExecutor) processTask(ctx context.Context, nodeID string, tmpl wfv1.Template) (*wfv1.NodeResult, time.Duration, error) {
	var executeTemplate templateExecutor
	switch {
	case tmpl.HTTP != nil:
		executeTemplate = ae.executeHTTPTemplate
	case tmpl.Plugin != nil:
		executeTemplate = ae.executePluginTemplate
	case tmpl.Resource != nil:
		executeTemplate = ae.executeResourceTemplate
	default:
		return nil, 0, fmt.Errorf("agent cannot execute: unknown task type: %v", tmpl.GetType())
	}
	result := &wfv1.NodeResult{}
	requeue, err := executeTemplate(ctx, nodeID, tmpl, result)
	if err != nil {
		result.Phase = wfv1.NodeFailed
		result.Message = err.Error()
	}
	return result, requeue, nil
}

func (ae *AgentExecutor) executeHTTPTemplate(ctx context.Context, _ string, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
	if tmpl.HTTP == nil {
		return 0, nil
	}
	// Read response.Body after cancel(), sometimes it return a context canceled error
	// For more detail  https://groups.google.com/g/golang-nuts/c/2FKwG6oEvos
	var cancel context.CancelFunc
	if tmpl.HTTP.TimeoutSeconds != nil {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*tmpl.HTTP.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	response, err := ae.executeHTTPTemplateRequest(ctx, tmpl.HTTP)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}

	outputs := wfv1.Outputs{Result: new(string(bodyBytes))}
	phase := wfv1.NodeSucceeded
	message := ""
	if tmpl.HTTP.SuccessCondition == "" {
		// Default success condition: StatusCode == 2xx
		success := response.StatusCode >= 200 && response.StatusCode < 300
		if !success {
			phase = wfv1.NodeFailed
			message = fmt.Sprintf("received non-2xx response code: %d", response.StatusCode)
		}
	} else {
		evalScope := map[string]any{
			"request": map[string]any{
				"method":    tmpl.HTTP.Method,
				"url":       tmpl.HTTP.URL,
				"body":      tmpl.HTTP.Body,
				"bodyBytes": tmpl.HTTP.GetBodyBytes(),
				"headers":   tmpl.HTTP.Headers.ToHeader(),
			},
			"response": map[string]any{
				"statusCode": response.StatusCode,
				"body":       string(bodyBytes),
				"headers":    response.Header,
			},
		}
		success, err := argoexpr.EvalBool(tmpl.HTTP.SuccessCondition, evalScope)
		if err != nil {
			return 0, err
		}
		if !success {
			phase = wfv1.NodeFailed
			message = fmt.Sprintf("successCondition '%s' evaluated false", tmpl.HTTP.SuccessCondition)
		}
	}

	result.Phase = phase
	result.Message = message
	result.Outputs = &outputs
	return 0, nil
}

var httpClientSkip = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

var httpClients = map[bool]*http.Client{
	false: http.DefaultClient,
	true:  httpClientSkip,
}

func (ae *AgentExecutor) executeHTTPTemplateRequest(ctx context.Context, httpTemplate *wfv1.HTTP) (*http.Response, error) {
	var (
		request *http.Request
		err     error
	)
	if httpTemplate.BodyFrom != nil {
		if httpTemplate.BodyFrom.Bytes != nil {
			request, err = http.NewRequestWithContext(ctx, httpTemplate.Method, httpTemplate.URL, bytes.NewBuffer(httpTemplate.BodyFrom.Bytes))
		}
	} else {
		request, err = http.NewRequestWithContext(ctx, httpTemplate.Method, httpTemplate.URL, bytes.NewBufferString(httpTemplate.Body))
	}
	if err != nil {
		return nil, err
	}

	for _, header := range httpTemplate.Headers {
		value := header.Value
		if header.ValueFrom != nil && header.ValueFrom.SecretKeyRef != nil {
			secret, secretErr := util.GetSecrets(ctx, ae.ClientSet, ae.Namespace, header.ValueFrom.SecretKeyRef.Name, header.ValueFrom.SecretKeyRef.Key)
			if secretErr != nil {
				return nil, secretErr
			}
			value = string(secret)
		}
		// for rewrite host header
		if strings.ToLower(header.Name) == "host" {
			request.Host = value
		} else {
			request.Header.Add(header.Name, value)
		}
	}

	response, err := httpClients[httpTemplate.InsecureSkipVerify].Do(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (ae *AgentExecutor) executePluginTemplate(ctx context.Context, _ string, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
	args := executorplugins.ExecuteTemplateArgs{
		Workflow: &executorplugins.Workflow{
			ObjectMeta: executorplugins.ObjectMeta{
				Name:      ae.WorkflowName,
				Namespace: ae.Namespace,
				UID:       ae.workflowUID,
			},
		},
		Template: &tmpl,
	}
	reply := &executorplugins.ExecuteTemplateReply{}
	for _, plug := range ae.plugins {
		if err := plug.ExecuteTemplate(ctx, args, reply); err != nil {
			return 0, err
		} else if reply.Node != nil {
			*result = *reply.Node
			if reply.Node.Phase == wfv1.NodeSucceeded {
				return 0, nil
			}
			return reply.GetRequeue(), nil
		}
	}
	return 0, fmt.Errorf("no plugin executed the template")
}

// injectMonitoredLabel stamps the manifest with the labels the agent uses
// to route informer events back to the originating task:
// common.LabelKeyMonitoredResource=workflowName for the watch selector,
// common.LabelKeyMonitoredResourceNodeID=nodeID for dispatch. Delete
// actions need no result watch and are returned unchanged.
func injectMonitoredLabel(manifest, action, workflowName, nodeID string) (string, error) {
	if action == "delete" {
		return manifest, nil
	}
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(manifest), &obj.Object); err != nil {
		return "", fmt.Errorf("parse manifest: %w", err)
	}
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[common.LabelKeyMonitoredResource] = workflowName
	labels[common.LabelKeyMonitoredResourceNodeID] = nodeID
	obj.SetLabels(labels)
	out, err := yaml.Marshal(obj.Object)
	if err != nil {
		return "", fmt.Errorf("serialize manifest: %w", err)
	}
	return string(out), nil
}

func (ae *AgentExecutor) obtainManifest(ctx context.Context, nodeID string, tmpl wfv1.Template) (string, string, error) {
	var raw string
	switch {
	case tmpl.Resource.Manifest != "":
		raw = tmpl.Resource.Manifest
	case tmpl.Resource.ManifestFrom != nil:
		// Resolve the manifest body off the input artifact named by
		// manifestFrom.artifact.name. Unlike the legacy resource pod, there's
		// no init container staging the file ahead of us — the agent has to
		// pull it from the artifact source itself.
		targetArtName := tmpl.Resource.ManifestFrom.Artifact.Name
		var art *wfv1.Artifact
		for i := range tmpl.Inputs.Artifacts {
			if tmpl.Inputs.Artifacts[i].Name == targetArtName {
				art = &tmpl.Inputs.Artifacts[i]
				break
			}
		}
		if art == nil {
			return "", "", fmt.Errorf("manifestFrom artifact %q not found in inputs.artifacts", targetArtName)
		}
		body, err := ae.downloadManifestArtifact(ctx, art)
		if err != nil {
			return "", "", err
		}
		raw = string(body)
	default:
		return "", "", fmt.Errorf("manifest was not supplied")
	}

	manifest, err := injectMonitoredLabel(raw, tmpl.Resource.Action, ae.WorkflowName, nodeID)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(manifest))
	hash := hex.EncodeToString(sum[:])
	path := filepath.Join("/tmp", fmt.Sprintf("manifest-%s.yaml", hash))
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		return "", "", fmt.Errorf("write manifest file: %w", err)
	}
	return manifest, path, nil
}

// downloadManifestArtifact fetches a single input artifact via the standard
// artifact-driver interface and returns its raw bytes.
//
// Archive support is intentionally narrow: tarballs (explicit or
// auto-detected) and zips (must be explicit, matching legacy behavior) are
// unarchived, but only when they contain exactly one file. Multi-file
// archives are rejected with a clear error pointing back to the legacy
// wait-pod path. This keeps label injection unambiguous — every monitored
// resource gets exactly one nodeID label — without giving up the common
// "tarball of one manifest YAML" case.
func (ae *AgentExecutor) downloadManifestArtifact(ctx context.Context, art *wfv1.Artifact) ([]byte, error) {
	if !art.HasLocationOrKey() {
		return nil, fmt.Errorf("manifest artifact %q has no location", art.Name)
	}
	driver, err := artifacts.NewDriver(ctx, art, ae)
	if err != nil {
		return nil, fmt.Errorf("init driver for artifact %q: %w", art.Name, err)
	}
	tmp, err := os.CreateTemp("", "agent-manifest-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if closeErr := tmp.Close(); closeErr != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("close temp file: %w", closeErr)
	}
	defer os.Remove(tmpPath)
	if loadErr := driver.Load(ctx, art, tmpPath); loadErr != nil {
		return nil, fmt.Errorf("load artifact %q: %w", art.Name, loadErr)
	}

	isTar, isZip, err := detectArchive(ctx, art, tmpPath)
	if err != nil {
		return nil, fmt.Errorf("detect archive for artifact %q: %w", art.Name, err)
	}
	if !isTar && !isZip {
		return os.ReadFile(tmpPath)
	}

	extractDir, err := os.MkdirTemp("", "agent-manifest-extract-*")
	if err != nil {
		return nil, fmt.Errorf("create extract dir: %w", err)
	}
	defer os.RemoveAll(extractDir)
	switch {
	case isTar:
		if extractErr := untar(tmpPath, extractDir); extractErr != nil {
			return nil, fmt.Errorf("untar artifact %q: %w", art.Name, extractErr)
		}
	case isZip:
		if extractErr := unzip(ctx, tmpPath, extractDir); extractErr != nil {
			return nil, fmt.Errorf("unzip artifact %q: %w", art.Name, extractErr)
		}
	}

	files, err := collectExtractedFiles(extractDir)
	if err != nil {
		return nil, fmt.Errorf("walk extracted artifact %q: %w", art.Name, err)
	}
	switch len(files) {
	case 0:
		return nil, fmt.Errorf("manifest artifact %q archive contained no files", art.Name)
	case 1:
		return os.ReadFile(files[0])
	default:
		return nil, fmt.Errorf("manifest artifact %q archive contains %d files; the agent monitor path only supports single-file archives — use an inline manifest or the legacy wait-pod path for multi-file manifestFrom", art.Name, len(files))
	}
}

// detectArchive mirrors the legacy unarchiveArtifact decision: respect an
// explicit Archive strategy when set, otherwise auto-detect tarballs by
// magic bytes (legacy behavior does not auto-detect zip).
func detectArchive(ctx context.Context, art *wfv1.Artifact, path string) (isTar, isZip bool, err error) {
	switch {
	case art.GetArchive().None != nil:
		return false, false, nil
	case art.GetArchive().Tar != nil:
		return true, false, nil
	case art.GetArchive().Zip != nil:
		return false, true, nil
	}
	isTar, err = isTarball(ctx, path)
	return isTar, false, err
}

func collectExtractedFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	})
	return files, err
}

// GetSecret implements workflow/artifacts/resource.Interface so the agent can
// supply credentials to artifact drivers.
func (ae *AgentExecutor) GetSecret(ctx context.Context, name, key string) (string, error) {
	b, err := util.GetSecrets(ctx, ae.ClientSet, ae.Namespace, name, key)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetConfigMapKey implements workflow/artifacts/resource.Interface.
func (ae *AgentExecutor) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	cm, err := ae.ClientSet.CoreV1().ConfigMaps(ae.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get configmap %q: %w", name, err)
	}
	v, ok := cm.Data[key]
	if !ok {
		return "", fmt.Errorf("configmap %q does not have key %q", name, key)
	}
	return v, nil
}

func getKubectlArguments(action string, manifestPath string, mergeStrategy string, flags []string) ([]string, error) {
	buff, err := os.ReadFile(filepath.Clean(manifestPath))
	if err != nil {
		return []string{}, argoerrors.New(argoerrors.CodeBadRequest, err.Error())
	}
	if len(buff) == 0 && len(flags) == 0 {
		return []string{}, argoerrors.New(argoerrors.CodeBadRequest, "Must provide at least one of flags or manifest.")
	}

	args := []string{
		"kubectl",
		action,
	}
	output := "json"

	if action == "delete" {
		args = append(args, "--ignore-not-found")
		output = "name"
	}

	appendFileFlag := true
	if action == "patch" {
		if mergeStrategy == "" {
			mergeStrategy = "strategic"
		}
		args = append(args, "--type")
		args = append(args, mergeStrategy)

		args = append(args, "--patch-file")
		args = append(args, manifestPath)

		// if there are flags and the manifest has no `kind`, assume: `kubectl patch <kind> <name> --patch-file <path>`
		// json patches also use patch files by definition and so require resource arguments
		// the other form in our case is `kubectl patch -f <path> --patch-file <path>`
		if mergeStrategy == "json" {
			appendFileFlag = false
		} else {
			var obj map[string]any
			err = yaml.Unmarshal(buff, &obj)
			if err != nil {
				return []string{}, argoerrors.New(argoerrors.CodeBadRequest, err.Error())
			}
			if len(flags) != 0 && obj["kind"] == nil {
				appendFileFlag = false
			}
		}
	}

	if len(flags) != 0 {
		args = append(args, flags...)
	}

	if len(buff) != 0 && appendFileFlag {
		args = append(args, "-f")
		args = append(args, manifestPath)
	}
	args = append(args, "-o")
	args = append(args, output)

	return args, nil
}

// inferGVR derives a GroupVersionResource from an unstructured object using
// the same lowercase-plural-namer heuristic kubectl applies under the hood.
func inferGVR(obj unstructured.Unstructured) schema.GroupVersionResource {
	gvk := obj.GroupVersionKind()
	lowercaseNamer := namer.NewAllLowercasePluralNamer(map[string]string{})
	plural := lowercaseNamer.Name(&gengotypes.Type{Name: gengotypes.Name{Name: gvk.Kind}})
	return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: plural}
}

func (ae *AgentExecutor) executeResource(ctx context.Context, action string, manifestPath string, mergeStrategy string, flags []string) (string, string, schema.GroupVersionResource, error) {
	args, err := getKubectlArguments(action, manifestPath, mergeStrategy, flags)
	if err != nil {
		return "", "", schema.GroupVersionResource{}, err
	}
	var out []byte
	err = retry.OnError(retry.DefaultBackoff, func(err error) bool {
		return errors.IsTransientErr(ctx, err)
	}, func() error {
		out, err = runKubectl(ctx, args...)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		var exErr *exec.ExitError
		if goerrors.As(err, &exErr) {
			errMsg := strings.TrimSpace(string(exErr.Stderr))
			err = argoerrors.Wrap(err, argoerrors.CodeBadRequest, errMsg)
		} else {
			err = argoerrors.Wrap(err, argoerrors.CodeBadRequest, err.Error())
		}
		return "", "", schema.GroupVersionResource{}, argoerrors.Wrap(err, argoerrors.CodeBadRequest, "no more retries "+err.Error())
	}
	if action == "delete" {
		return "", "", schema.GroupVersionResource{}, nil
	}
	if action == "get" && len(out) == 0 {
		return "", "", schema.GroupVersionResource{}, nil
	}
	obj := unstructured.Unstructured{}
	err = json.Unmarshal(out, &obj)
	if err != nil {
		return "", "", schema.GroupVersionResource{}, err
	}
	resourceGroup := obj.GroupVersionKind().Group
	resourceName := obj.GetName()
	resourceKind := obj.GroupVersionKind().Kind
	if resourceName == "" || resourceKind == "" {
		return "", "", schema.GroupVersionResource{}, argoerrors.New(argoerrors.CodeBadRequest, "Kind and name are both required but at least one of them is missing from the manifest")
	}
	resourceFullName := fmt.Sprintf("%s.%s/%s", strings.ToLower(resourceKind), resourceGroup, resourceName)
	gvr := inferGVR(obj)
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{"namespace": obj.GetNamespace(), "resource": resourceFullName, "gvr": gvr.String()}).Info(ctx, "Resource")

	return obj.GetNamespace(), resourceName, gvr, nil
}

// handleDone is the per-event callback fed to the MonitoredResourceInformer.
// It dispatches events to in-flight resource template tasks by reading the
// node-ID label off the object, evaluates success/failure conditions, and
// posts a terminal NodeResult to the responseQueue when a condition fires.
//
// Runs on informer goroutines, so it must not call back into anything that
// blocks on the task workers.
func (ae *AgentExecutor) handleDone(ctx context.Context, obj *unstructured.Unstructured, deleted bool) {
	logger := logging.RequireLoggerFromContext(ctx).
		WithField("function", "handleDone").
		WithField("resource", fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName()))

	nodeID := obj.GetLabels()[common.LabelKeyMonitoredResourceNodeID]
	if nodeID == "" {
		logger.Info(ctx, "Skipping event: object missing monitored-resource node-ID label")
		return
	}
	logger = logger.WithField("nodeID", nodeID)

	ae.pendingMu.Lock()
	pending, ok := ae.pendingTasks[nodeID]
	ae.pendingMu.Unlock()
	if !ok {
		logger.Info(ctx, "Skipping event: node already resolved or never registered")
		return
	}

	if deleted {
		logger.Info(ctx, "Monitored resource deleted before completion")
		ae.completeNode(nodeID, &wfv1.NodeResult{
			Phase:   wfv1.NodeFailed,
			Message: "monitored resource was deleted before completion",
		})
		return
	}

	jsonBytes, err := obj.MarshalJSON()
	if err != nil {
		logger.WithError(err).Warn(ctx, "Failed to marshal monitored object; waiting for next event")
		return
	}

	retry, matchErr := matchConditions(ctx, jsonBytes, pending.successReqs, pending.failReqs)
	if retry {
		logger.Info(ctx, "Conditions not yet matched; waiting for next event")
		return
	}
	if matchErr != nil {
		logger.WithError(matchErr).Info(ctx, "Failure condition matched; completing node as failed")
		ae.completeNode(nodeID, &wfv1.NodeResult{
			Phase:   wfv1.NodeFailed,
			Message: matchErr.Error(),
		})
		return
	}

	outs, err := extractOutputs(ctx, jsonBytes, pending.outputs)
	if err != nil {
		logger.WithError(err).Info(ctx, "Failed to extract outputs; completing node as failed")
		ae.completeNode(nodeID, &wfv1.NodeResult{
			Phase:   wfv1.NodeFailed,
			Message: fmt.Sprintf("extract outputs: %v", err),
		})
		return
	}
	result := &wfv1.NodeResult{Phase: wfv1.NodeSucceeded}
	if len(outs) > 0 {
		result.Outputs = &wfv1.Outputs{Parameters: outs}
	}
	logger.Info(ctx, "Success condition matched; completing node as succeeded")
	ae.completeNode(nodeID, result)
}

// completeNode drops the pending entry and pushes the result onto the
// responseQueue for the patchWorker.
func (ae *AgentExecutor) completeNode(nodeID string, result *wfv1.NodeResult) {
	ae.pendingMu.Lock()
	delete(ae.pendingTasks, nodeID)
	ae.pendingMu.Unlock()
	ae.responseQueue <- response{NodeID: nodeID, Result: result}
}

// extractOutputs resolves each output parameter's ValueFrom against the
// resource JSON: JSONPath via client-go's jsonpath, JQFilter via gojq,
// Default as a fallback. Parameters with no ValueFrom are passed through
// unchanged.
func extractOutputs(ctx context.Context, jsonBytes []byte, params []wfv1.Parameter) ([]wfv1.Parameter, error) {
	if len(params) == 0 {
		return nil, nil
	}
	var data map[string]any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("unmarshal object: %w", err)
	}
	out := make([]wfv1.Parameter, 0, len(params))
	for _, p := range params {
		np := p
		if p.ValueFrom == nil {
			out = append(out, np)
			continue
		}
		var v string
		switch {
		case p.ValueFrom.JSONPath != "":
			tpl := p.ValueFrom.JSONPath
			if !strings.HasPrefix(tpl, "{") {
				tpl = "{" + tpl + "}"
			}
			j := jsonpath.New(p.Name)
			j.AllowMissingKeys(true)
			if err := j.Parse(tpl); err != nil {
				return nil, fmt.Errorf("parameter %q jsonPath parse: %w", p.Name, err)
			}
			buf := &bytes.Buffer{}
			if err := j.Execute(buf, data); err != nil {
				return nil, fmt.Errorf("parameter %q jsonPath execute: %w", p.Name, err)
			}
			v = buf.String()
		case p.ValueFrom.JQFilter != "":
			r, err := jqFilter(ctx, jsonBytes, p.ValueFrom.JQFilter)
			if err != nil {
				return nil, fmt.Errorf("parameter %q jqFilter: %w", p.Name, err)
			}
			v = r
		default:
			if p.ValueFrom.Default != nil {
				v = p.ValueFrom.Default.String()
			}
		}
		np.Value = wfv1.AnyStringPtr(v)
		out = append(out, np)
	}
	return out, nil
}

func (ae *AgentExecutor) executeResourceTemplate(ctx context.Context, nodeID string, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	// find out the if its a resource
	if tmpl.Resource == nil {
		return 0, fmt.Errorf("expected a resource template")
	}

	_, manifestPath, err := ae.obtainManifest(ctx, nodeID, tmpl)
	if err != nil {
		return 0, err
	}

	action := tmpl.Resource.Action
	isDelete := action == "delete"

	if isDelete && (tmpl.Resource.SuccessCondition != "" || tmpl.Resource.FailureCondition != "" || len(tmpl.Outputs.Parameters) > 0) {
		return 0, fmt.Errorf("successCondition, failureCondition and outputs are not supported for delete action")
	}

	_, _, gvr, err := ae.executeResource(ctx, action, manifestPath, tmpl.Resource.MergeStrategy, tmpl.Resource.Flags)
	if err != nil {
		return 0, err
	}
	if isDelete {
		result.Phase = wfv1.NodeSucceeded
		return 0, nil
	}

	successReqs, failReqs, err := parseResourceConditions(tmpl.Resource.SuccessCondition, tmpl.Resource.FailureCondition)
	if err != nil {
		return 0, err
	}
	ae.pendingMu.Lock()
	ae.pendingTasks[nodeID] = pendingResourceTask{
		successReqs: successReqs,
		failReqs:    failReqs,
		outputs:     tmpl.Outputs.Parameters,
	}
	ae.pendingMu.Unlock()

	if err := ae.resourceInformer.Watch(ctx, gvr); err != nil {
		logger.WithError(err).Info(ctx, "was unable to watch on the resource")
		ae.completeNode(nodeID, &wfv1.NodeResult{
			Phase:   wfv1.NodeFailed,
			Message: fmt.Sprintf("watch %s: %v", gvr, err),
		})
		return 0, nil
	}
	result.Phase = wfv1.NodeRunning
	return 0, nil
}

func parseResourceConditions(success, failure string) (labels.Requirements, labels.Requirements, error) {
	var successReqs, failReqs labels.Requirements
	if success != "" {
		sel, err := labels.Parse(success)
		if err != nil {
			return nil, nil, fmt.Errorf("parse successCondition %q: %w", success, err)
		}
		successReqs, _ = sel.Requirements()
	}
	if failure != "" {
		sel, err := labels.Parse(failure)
		if err != nil {
			return nil, nil, fmt.Errorf("parse failureCondition %q: %w", failure, err)
		}
		failReqs, _ = sel.Requirements()
	}
	return successReqs, failReqs, nil
}

func IsWorkflowCompleted(wts *wfv1.WorkflowTaskSet) bool {
	return wts.Labels[common.LabelKeyCompleted] == "true"
}
