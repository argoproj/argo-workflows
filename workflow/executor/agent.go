package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	argohttp "github.com/argoproj/argo-workflows/v3/workflow/executor/http"
)

type AgentExecutor struct {
	WorkflowName      string
	ClientSet         kubernetes.Interface
	WorkflowInterface workflow.Interface
	RESTClient        rest.Interface
	Namespace         string
	ConsideredTasks   map[string]bool
}

type patchResponse struct {
	NodeId string
	Patch  []byte
}

func (ae *AgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	taskSetInterface := ae.WorkflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(ae.Namespace)
	patches := make(chan patchResponse)
	go func() {
		for patch := range patches {
			log.WithFields(log.Fields{"workflow": ae.WorkflowName, "nodeID": patch.NodeId}).Error("SIMON Processing Patch")

			obj, err := taskSetInterface.Patch(ctx, ae.WorkflowName, types.MergePatchType, patch.Patch, metav1.PatchOptions{})
			if err != nil {
				log.WithError(err).WithField("taskset", obj).Errorf("TaskSet Patch Failed")
			} else {
				log.WithField("taskset", obj).Infof("SIMON Patched TaskSet, %s", patch)
			}
		}
	}()

	for {
		wfWatch, err := taskSetInterface.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + ae.WorkflowName})
		if err != nil {
			return err
		}

		for event := range wfWatch.ResultChan() {
			log.WithFields(log.Fields{"workflow": ae.WorkflowName, "event_type": event.Type}).Infof("SIMON TaskSet Event")

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
				nodeID, tmpl := nodeID, tmpl
				go func() {
					log.WithFields(log.Fields{"nodeID": nodeID}).Error("SIMON Attempting task")

					// Do not work on tasks that have already been considered once, to prevent unintentional double hitting
					// of endpoints
					if _, ok := ae.ConsideredTasks[nodeID]; ok {
						log.WithFields(log.Fields{"nodeID": nodeID}).Error("SIMON Task is already considered")
						return
					}

					ae.ConsideredTasks[nodeID] = true

					log.WithFields(log.Fields{"nodeID": nodeID}).Error("SIMON Processing task")
					patch, err := ae.processTask(ctx, nodeID, tmpl)
					if err != nil {
						log.WithFields(log.Fields{"error": err, "nodeID": nodeID}).Error("Error in agent task")
						return
					}

					log.WithFields(log.Fields{"nodeID": nodeID}).Error("SIMON Sending patch")
					patches <- patchResponse{NodeId: nodeID, Patch: patch}
				}()
			}
		}
	}
}

func (ae *AgentExecutor) processTask(ctx context.Context, nodeID string, tmpl wfv1.Template) ([]byte, error) {
	switch {
	case tmpl.HTTP != nil:
		result := wfv1.NodeResult{}
		if outputs, err := ae.executeHTTPTemplate(ctx, tmpl); err != nil {
			result.Phase = wfv1.NodeFailed
			result.Message = err.Error()
		} else {
			result.Phase = wfv1.NodeSucceeded
			result.Outputs = outputs
		}
		nodeResults := map[string]wfv1.NodeResult{}
		nodeResults[nodeID] = result

		patch, err := json.Marshal(map[string]interface{}{"status": wfv1.WorkflowTaskSetStatus{Nodes: nodeResults}})
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}

		return patch, nil
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
