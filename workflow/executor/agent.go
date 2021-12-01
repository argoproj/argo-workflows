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
	WorkflowName      string
	ClientSet         kubernetes.Interface
	WorkflowInterface workflow.Interface
	RESTClient        rest.Interface
	Namespace         string
	consideredTasks   map[string]bool
}

func NewAgentExecutor(clientSet kubernetes.Interface, restClient rest.Interface, config *rest.Config, namespace, workflowName string) *AgentExecutor {
	return &AgentExecutor{
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

const (
	EnvAgentTaskWorkers = "ARGO_AGENT_TASK_WORKERS"
	EnvAgentPatchRate   = "ARGO_AGENT_PATCH_RATE"
)

func (ae *AgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	taskWorkers := env.LookupEnvIntOr(EnvAgentTaskWorkers, 16)
	requeueTime := env.LookupEnvDurationOr(EnvAgentPatchRate, 10*time.Second)
	log.WithFields(log.Fields{"taskWorkers": taskWorkers, "requeueTime": requeueTime}).Info("Starting Agent")

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
			log.WithFields(log.Fields{"workflow": ae.WorkflowName, "event_type": event.Type}).Infof("TaskSet Event")

			if event.Type == watch.Deleted {
				// We're done if the task set is deleted
				return nil
			}

			taskSet, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			if IsWorkflowCompleted(taskSet) {
				log.WithField("workflow", ae.WorkflowName).Info("Workflow completed... stopping agent")
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
		log.WithFields(log.Fields{"nodeID": nodeID}).Info("Attempting task")

		// Do not work on tasks that have already been considered once, to prevent calling an endpoint more
		// than once unintentionally.
		if _, ok := ae.consideredTasks[nodeID]; ok {
			log.WithFields(log.Fields{"nodeID": nodeID}).Info("Task is already considered")
			continue
		}

		ae.consideredTasks[nodeID] = true

		log.WithFields(log.Fields{"nodeID": nodeID}).Info("Processing task")
		result, err := ae.processTask(ctx, tmpl)
		if err != nil {
			log.WithFields(log.Fields{"error": err, "nodeID": nodeID}).Error("Error in agent task")
			return
		}

		log.WithFields(log.Fields{"nodeID": nodeID}).Info("Sending result")
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
				log.WithError(err).Error("Generating Patch Failed")
				continue
			}

			log.WithFields(log.Fields{"workflow": ae.WorkflowName}).Info("Processing Patch")

			obj, err := taskSetInterface.Patch(ctx, ae.WorkflowName, types.MergePatchType, patch, metav1.PatchOptions{})
			if err != nil {
				isTransientErr := errors.IsTransientErr(err)
				log.WithError(err).WithFields(log.Fields{"taskset": obj, "is_transient_error": isTransientErr}).Errorf("TaskSet Patch Failed")

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

			log.WithField("taskset", obj).Infof("Patched TaskSet")
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
	outputs.Parameters = append(outputs.Parameters, wfv1.Parameter{Name: "result", Value: wfv1.AnyStringPtr(response)})

	return outputs, nil
}

func IsWorkflowCompleted(wts *wfv1.WorkflowTaskSet) bool {
	return wts.Labels[common.LabelKeyCompleted] == "true"
}
