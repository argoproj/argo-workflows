package executor

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type AgentExecutor struct {
	LabelSelector     string
	WorkflowName      string
	ClientSet         kubernetes.Interface
	WorkflowInterface workflow.Interface
	RESTClient        rest.Interface
	Namespace         string
	consideredTasks   *sync.Map
	plugins           []executorplugins.TemplateExecutor
}

type templateExecutor = func(ctx context.Context, workflowName string, workflowUID string, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error)

func NewAgentExecutor(clientSet kubernetes.Interface, restClient rest.Interface, config *rest.Config, namespace, labelSelector, workflowName string, plugins []executorplugins.TemplateExecutor) *AgentExecutor {
	return &AgentExecutor{
		ClientSet:         clientSet,
		RESTClient:        restClient,
		Namespace:         namespace,
		LabelSelector:     labelSelector,
		WorkflowName:      workflowName,
		WorkflowInterface: workflow.NewForConfigOrDie(config),
		consideredTasks:   &sync.Map{},
		plugins:           plugins,
	}
}

type task struct {
	NodeID      string
	Template    wfv1.Template
	TaskSetName string
	WorkflowUID string
}

type response struct {
	NodeID      string
	Result      *wfv1.NodeResult
	TaskSetName string
}

func (ae *AgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrashWithContext(ctx, runtimeutil.PanicHandlers...)

	taskWorkers := env.LookupEnvIntOr(ctx, common.EnvAgentTaskWorkers, 16)
	requeueTime := env.LookupEnvDurationOr(ctx, common.EnvAgentPatchRate, 10*time.Second)
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("labelSelector", ae.LabelSelector).
		WithField("taskWorkers", taskWorkers).
		WithField("requeueTime", requeueTime).
		Info(ctx, "Starting Agent")

	taskQueue := make(chan task)
	responseQueue := make(chan response)
	taskSetInterface := ae.WorkflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(ae.Namespace)

	go ae.patchWorker(ctx, taskSetInterface, responseQueue, requeueTime)
	for range taskWorkers {
		go ae.taskWorker(ctx, taskQueue, responseQueue)
	}

	for {
		// Use label selector from environment variable (set by controller) or workflow name
		var wfWatch watch.Interface
		var err error
		if ae.LabelSelector == "" {
			wfWatch, err = taskSetInterface.Watch(ctx, metav1.ListOptions{
				FieldSelector: "metadata.name=" + ae.WorkflowName,
			})
		} else {
			wfWatch, err = taskSetInterface.Watch(ctx, metav1.ListOptions{
				LabelSelector: ae.LabelSelector,
			})
		}

		if err != nil {
			return err
		}

		for event := range wfWatch.ResultChan() {
			logger.WithField("event_type", event.Type).Info(ctx, "TaskSet Event")

			if event.Type == watch.Deleted {
				// TaskSet deleted, but continue watching for others
				continue
			}

			taskSet, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}

			taskSetName := taskSet.Name

			if IsWorkflowCompleted(taskSet) {
				logger.WithField("taskSet", taskSetName).Info(ctx, "Workflow completed, skipping tasks")
				continue
			}

			// Extract workflow UID from owner references
			var workflowUID string
			if len(taskSet.OwnerReferences) > 0 {
				workflowUID = string(taskSet.OwnerReferences[0].UID)
			}

			for nodeID, tmpl := range taskSet.Spec.Tasks {
				taskQueue <- task{
					NodeID:      nodeID,
					Template:    tmpl,
					TaskSetName: taskSetName,
					WorkflowUID: workflowUID,
				}
			}
		}
	}
}

func (ae *AgentExecutor) taskWorker(ctx context.Context, taskQueue chan task, responseQueue chan response) {
	for {
		task, ok := <-taskQueue
		if !ok {
			break
		}
		nodeID, tmpl := task.NodeID, task.Template
		ctx, logger := logging.RequireLoggerFromContext(ctx).WithField("nodeID", nodeID).InContext(ctx)

		// Do not work on tasks that have already been considered once, to prevent calling an endpoint more
		// than once unintentionally.
		if _, ok := ae.consideredTasks.LoadOrStore(nodeID, true); ok {
			logger.Info(ctx, "Task is already considered")
			continue
		}

		logger.Info(ctx, "Processing task")
		result, requeue, err := ae.processTask(ctx, task.TaskSetName, task.WorkflowUID, tmpl)
		if err != nil {
			logger.WithError(err).Error(ctx, "Error in agent task")
			result = &wfv1.NodeResult{
				Phase:   wfv1.NodeError,
				Message: err.Error(),
			}
		}

		logger.
			WithField("phase", result.Phase).
			WithField("message", result.Message).
			WithField("requeue", requeue).
			Info(ctx, "Sending result")

		if result.Phase != "" {
			responseQueue <- response{
				NodeID:      nodeID,
				Result:      result,
				TaskSetName: task.TaskSetName,
			}
		}

		if requeue > 0 {
			time.AfterFunc(requeue, func() {
				ae.consideredTasks.Delete(nodeID)
				taskQueue <- task
			})
		}
	}
}

func (ae *AgentExecutor) patchWorker(ctx context.Context, taskSetInterface v1alpha1.WorkflowTaskSetInterface, responseQueue chan response, requeueTime time.Duration) {
	ticker := time.NewTicker(requeueTime)
	defer ticker.Stop()

	taskSetResults := make(map[string]map[string]wfv1.NodeResult)

	logger := logging.RequireLoggerFromContext(ctx)
	for {
		select {
		case res := <-responseQueue:
			if taskSetResults[res.TaskSetName] == nil {
				taskSetResults[res.TaskSetName] = make(map[string]wfv1.NodeResult)
			}
			taskSetResults[res.TaskSetName][res.NodeID] = *res.Result

		case <-ticker.C:
			if len(taskSetResults) == 0 {
				continue
			}

			for taskSetName, nodeResults := range taskSetResults {
				patch, err := json.Marshal(map[string]any{"status": wfv1.WorkflowTaskSetStatus{Nodes: nodeResults}})
				if err != nil {
					logger.WithError(err).WithField("taskSet", taskSetName).Error(ctx, "Generating Patch Failed")
					continue
				}

				logger.WithField("taskSet", taskSetName).
					WithField("nodeCount", len(nodeResults)).
					Info(ctx, "Processing Patch")

				err = retry.OnError(wait.Backoff{
					Duration: time.Second,
					Factor:   2,
					Jitter:   0.1,
					Steps:    5,
					Cap:      30 * time.Second,
				}, func(err error) bool {
					return errors.IsTransientErr(ctx, err)
				}, func() error {
					_, err := taskSetInterface.Patch(ctx, taskSetName, types.MergePatchType, patch, metav1.PatchOptions{}, "status")
					return err
				})

				if err != nil && !errors.IsTransientErr(ctx, err) {
					logger.WithError(err).WithField("taskSet", taskSetName).
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

				logger.WithField("taskSet", taskSetName).Info(ctx, "Patched TaskSet")
			}

			taskSetResults = make(map[string]map[string]wfv1.NodeResult)
		}
	}
}

func (ae *AgentExecutor) processTask(ctx context.Context, workflowName string, workflowUID string, tmpl wfv1.Template) (*wfv1.NodeResult, time.Duration, error) {
	var executeTemplate templateExecutor
	switch {
	case tmpl.HTTP != nil:
		executeTemplate = ae.executeHTTPTemplate
	case tmpl.Plugin != nil:
		executeTemplate = ae.executePluginTemplate
	default:
		return nil, 0, fmt.Errorf("agent cannot execute: unknown task type: %v", tmpl.GetType())
	}
	result := &wfv1.NodeResult{}
	requeue, err := executeTemplate(ctx, workflowName, workflowUID, tmpl, result)
	if err != nil {
		result.Phase = wfv1.NodeFailed
		result.Message = err.Error()
	}
	return result, requeue, nil
}

func (ae *AgentExecutor) executeHTTPTemplate(ctx context.Context, workflowName string, workflowUID string, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
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

	outputs := wfv1.Outputs{Result: ptr.To(string(bodyBytes))}
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
			secret, err := util.GetSecrets(ctx, ae.ClientSet, ae.Namespace, header.ValueFrom.SecretKeyRef.Name, header.ValueFrom.SecretKeyRef.Key)
			if err != nil {
				return nil, err
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

func (ae *AgentExecutor) executePluginTemplate(ctx context.Context, workflowName string, workflowUID string, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
	args := executorplugins.ExecuteTemplateArgs{
		Workflow: &executorplugins.Workflow{
			ObjectMeta: executorplugins.ObjectMeta{
				Name:      workflowName,
				Namespace: ae.Namespace,
				UID:       workflowUID,
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

func IsWorkflowCompleted(wts *wfv1.WorkflowTaskSet) bool {
	return wts.Labels[common.LabelKeyCompleted] == "true"
}
