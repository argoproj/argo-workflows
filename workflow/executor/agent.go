package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	argohttp "github.com/argoproj/argo-workflows/v3/workflow/executor/http"
)

type AgentExecutor struct {
	log               *log.Entry
	WorkflowName      string
	ClientSet         kubernetes.Interface
	WorkflowInterface workflow.Interface
	RESTClient        rest.Interface
	Namespace         string
	consideredTasks   map[string]bool
}

func NewAgentExecutor(clientSet kubernetes.Interface, restClient rest.Interface, config *rest.Config, namespace, workflowName string) *AgentExecutor {
	return &AgentExecutor{
		log:               log.WithField("workflow", workflowName),
		ClientSet:         clientSet,
		RESTClient:        restClient,
		Namespace:         namespace,
		WorkflowName:      workflowName,
		WorkflowInterface: workflow.NewForConfigOrDie(config),
		consideredTasks:   make(map[string]bool),
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
	for task := range taskQueue {
		nodeID, tmpl := task.NodeId, task.Template
		log := log.WithField("nodeID", nodeID)

		// Do not work on tasks that have already been considered once, to prevent calling an endpoint more
		// than once unintentionally.
		if _, ok := ae.consideredTasks[nodeID]; ok {
			log.Info("Task is already considered")
			continue
		}

		ae.consideredTasks[nodeID] = true

		log.Info("Processing task")
		result, err := ae.processTask(ctx, tmpl)
		if err != nil {
			log.WithError(err).Error("Error in agent task")
			return
		}

		log.WithField("phase", result.Phase).
			WithField("message", result.Message).
			Info("Sending result")
		responseQueue <- response{NodeId: nodeID, Result: result}
	}
}

func (ae *AgentExecutor) patchWorker(ctx context.Context, taskSetInterface v1alpha1.WorkflowTaskSetInterface, responseQueue chan response, requeueTime time.Duration) {
	ticker := time.NewTicker(requeueTime)
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

			_, err = taskSetInterface.Patch(ctx, ae.WorkflowName, types.MergePatchType, patch, metav1.PatchOptions{})
			if err != nil {
				isTransientErr := errors.IsTransientErr(err)
				ae.log.WithError(err).
					WithField("is_transient_error", isTransientErr).
					Error("TaskSet Patch Failed")

				// If this is not a transient error, then it's likely that the contents of the patch have caused the error.
				// To avoid a deadlock with the workflow overall, or an infinite loop, fail and propagate the error messages
				// to the nodes.
				// If this is a transient error, then simply do nothing and another patch will be retried in the next tick.
				if !isTransientErr {
					for node := range nodeResults {
						nodeResults[node] = wfv1.NodeResult{
							Phase:   wfv1.NodeError,
							Message: fmt.Sprintf("HTTP request completed successfully but an error occurred when patching its result: %s", err),
						}
					}
				}
				continue
			}

			// Patch was successful, clear nodeResults for next iteration
			nodeResults = map[string]wfv1.NodeResult{}

			ae.log.Info("Patched TaskSet")
		}
	}
}

func (ae *AgentExecutor) processTask(ctx context.Context, tmpl wfv1.Template) (*wfv1.NodeResult, error) {
	switch {
	case tmpl.HTTP != nil:
		var result wfv1.NodeResult
		if outputs, err := ae.executeHTTPTemplate(ctx, tmpl); err != nil {
			result.Phase = wfv1.NodeFailed
			result.Message = err.Error()
		} else {
			result.Phase = wfv1.NodeSucceeded
			result.Outputs = outputs
		}
		return &result, nil
	default:
		return nil, fmt.Errorf("agent cannot execute: unknown task type")
	}
}

func (ae *AgentExecutor) executeHTTPTemplate(ctx context.Context, tmpl wfv1.Template) (*wfv1.Outputs, error) {
	if tmpl.HTTP == nil {
		return nil, fmt.Errorf("attempting to execute template that is not of type HTTP")
	}
	httpTemplate := tmpl.HTTP
	request, err := http.NewRequest(httpTemplate.Method, httpTemplate.URL, bytes.NewBufferString(httpTemplate.Body))
	if err != nil {
		return nil, err
	}
	request = request.WithContext(ctx)

	for _, header := range httpTemplate.Headers {
		value := header.Value
		if header.ValueFrom != nil && header.ValueFrom.SecretKeyRef != nil {
			secret, err := util.GetSecrets(ctx, ae.ClientSet, ae.Namespace, header.ValueFrom.SecretKeyRef.Name, header.ValueFrom.SecretKeyRef.Key)
			if err != nil {
				return nil, err
			}
			value = string(secret)
		}
		request.Header.Add(header.Name, value)
	}
	response, err := argohttp.SendHttpRequest(request, httpTemplate.TimeoutSeconds)
	if err != nil {
		return nil, err
	}
	outputs := &wfv1.Outputs{}
	outputs.Result = pointer.StringPtr(response)

	return outputs, nil
}

func IsWorkflowCompleted(wts *wfv1.WorkflowTaskSet) bool {
	return wts.Labels[common.LabelKeyCompleted] == "true"
}
