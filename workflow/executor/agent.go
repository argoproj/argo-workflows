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

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type AgentExecutor struct {
	log               *log.Entry
	WorkflowName      string
	workflowUID       string
	ClientSet         kubernetes.Interface
	WorkflowInterface workflow.Interface
	RESTClient        rest.Interface
	Namespace         string
	consideredTasks   *sync.Map
	plugins           []executorplugins.TemplateExecutor
}

type templateExecutor = func(ctx context.Context, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error)

func NewAgentExecutor(clientSet kubernetes.Interface, restClient rest.Interface, config *rest.Config, namespace, workflowName, workflowUID string, plugins []executorplugins.TemplateExecutor) *AgentExecutor {
	return &AgentExecutor{
		log:               log.WithField("workflow", workflowName),
		ClientSet:         clientSet,
		RESTClient:        restClient,
		Namespace:         namespace,
		WorkflowName:      workflowName,
		workflowUID:       workflowUID,
		WorkflowInterface: workflow.NewForConfigOrDie(config),
		consideredTasks:   &sync.Map{},
		plugins:           plugins,
	}
}

type task struct {
	NodeId   string
	Template wfv1.Template
}

type response struct {
	NodeId string
	Result *wfv1.NodeResult
}

func (ae *AgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	taskWorkers := env.LookupEnvIntOr(common.EnvAgentTaskWorkers, 16)
	requeueTime := env.LookupEnvDurationOr(common.EnvAgentPatchRate, 10*time.Second)
	ae.log.WithFields(log.Fields{"taskWorkers": taskWorkers, "requeueTime": requeueTime}).Info("Starting Agent")

	taskQueue := make(chan task)
	responseQueue := make(chan response)
	taskSetInterface := ae.WorkflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(ae.Namespace)

	go ae.patchWorker(ctx, taskSetInterface, responseQueue, requeueTime)
	for i := 0; i < taskWorkers; i++ {
		go ae.taskWorker(ctx, taskQueue, responseQueue)
	}

	for {
		wfWatch, err := taskSetInterface.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + ae.WorkflowName})
		if err != nil {
			return err
		}

		for event := range wfWatch.ResultChan() {
			ae.log.WithField("event_type", event.Type).Info("TaskSet Event")

			if event.Type == watch.Deleted {
				// We're done if the task set is deleted
				return nil
			}

			taskSet, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			if IsWorkflowCompleted(taskSet) {
				ae.log.Info("Workflow completed... stopping agent")
				return nil
			}

			for nodeID, tmpl := range taskSet.Spec.Tasks {
				taskQueue <- task{NodeId: nodeID, Template: tmpl}
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
		nodeID, tmpl := task.NodeId, task.Template
		log := log.WithField("nodeID", nodeID)

		// Do not work on tasks that have already been considered once, to prevent calling an endpoint more
		// than once unintentionally.
		if _, ok := ae.consideredTasks.LoadOrStore(nodeID, true); ok {
			log.Info("Task is already considered")
			continue
		}

		log.Info("Processing task")
		result, requeue, err := ae.processTask(ctx, tmpl)
		if err != nil {
			log.WithError(err).Error("Error in agent task")
			result = &wfv1.NodeResult{
				Phase:   wfv1.NodeError,
				Message: fmt.Sprintf("error processing task: %s", err),
			}
			// Do not return or continue here, the "errored" result still needs to be propagated to the responseQueue below
		}

		log.
			WithField("phase", result.Phase).
			WithField("message", result.Message).
			WithField("requeue", requeue).
			Info("Sending result")

		if result.Phase != "" {
			responseQueue <- response{NodeId: nodeID, Result: result}
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
	nodeResults := map[string]wfv1.NodeResult{}
	for {
		select {
		case res := <-responseQueue:
			nodeResults[res.NodeId] = *res.Result
		case <-ticker.C:
			if len(nodeResults) == 0 {
				continue
			}

			patch, err := json.Marshal(map[string]interface{}{"status": wfv1.WorkflowTaskSetStatus{Nodes: nodeResults}})
			if err != nil {
				ae.log.WithError(err).Error("Generating Patch Failed")
				continue
			}

			ae.log.Info("Processing Patch")

			err = retry.OnError(wait.Backoff{
				Duration: time.Second,
				Factor:   2,
				Jitter:   0.1,
				Steps:    5,
				Cap:      30 * time.Second,
			}, errors.IsTransientErr, func() error {
				_, err := taskSetInterface.Patch(ctx, ae.WorkflowName, types.MergePatchType, patch, metav1.PatchOptions{}, "status")
				return err
			})

			if err != nil && !errors.IsTransientErr(err) {
				ae.log.WithError(err).
					Error("TaskSet Patch Failed")

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

			log.Info("Patched TaskSet")
		}
	}
}

func (ae *AgentExecutor) processTask(ctx context.Context, tmpl wfv1.Template) (*wfv1.NodeResult, time.Duration, error) {
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
	requeue, err := executeTemplate(ctx, tmpl, result)
	if err != nil {
		result.Phase = wfv1.NodeFailed
		result.Message = err.Error()
	}
	return result, requeue, nil
}

func (ae *AgentExecutor) executeHTTPTemplate(ctx context.Context, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
	if tmpl.HTTP == nil {
		return 0, nil
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

	outputs := wfv1.Outputs{Result: pointer.StringPtr(string(bodyBytes))}
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
		evalScope := map[string]interface{}{
			"request": map[string]interface{}{
				"method":    tmpl.HTTP.Method,
				"url":       tmpl.HTTP.URL,
				"body":      tmpl.HTTP.Body,
				"bodyBytes": tmpl.HTTP.GetBodyBytes(),
				"headers":   tmpl.HTTP.Headers.ToHeader(),
			},
			"response": map[string]interface{}{
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
			request, err = http.NewRequest(httpTemplate.Method, httpTemplate.URL, bytes.NewBuffer(httpTemplate.BodyFrom.Bytes))
		}
	} else {
		request, err = http.NewRequest(httpTemplate.Method, httpTemplate.URL, bytes.NewBufferString(httpTemplate.Body))
	}
	if err != nil {
		return nil, err
	}

	if httpTemplate.TimeoutSeconds != nil {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(*httpTemplate.TimeoutSeconds)*time.Second)
		defer cancel()
		request = request.WithContext(ctx)
	} else {
		request = request.WithContext(ctx)
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

func (ae *AgentExecutor) executePluginTemplate(ctx context.Context, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
	args := executorplugins.ExecuteTemplateArgs{
		Workflow: &executorplugins.Workflow{
			ObjectMeta: executorplugins.ObjectMeta{
				Name:      ae.WorkflowName,
				Namespace: ae.Namespace,
				Uid:       ae.workflowUID,
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
