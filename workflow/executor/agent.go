package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/argoproj/pkg/sync"
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
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
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
	CompletedTasks    map[string]struct{}
	KeyLock           sync.KeyLock
}

type patchResponse struct {
	NodeId string
	Patch  []byte
}

func (ae *AgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	taskSetInterface := ae.WorkflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(ae.Namespace)
	for {
		wfWatch, err := taskSetInterface.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + ae.WorkflowName})
		if err != nil {
			return err
		}

		patches := make(chan patchResponse)
		go func() {
			for patch := range patches {
				log.WithFields(log.Fields{"nodeID": patch.NodeId}).Error("SIMON Processing patch")
				ae.processResult(ctx, patch.NodeId, patch.Patch, taskSetInterface)
			}
		}()

		for event := range wfWatch.ResultChan() {
			log.WithField("taskset", ae.WorkflowName).Infof("SIMON watching taskset, %v", event)

			if event.Type == watch.Deleted {
				// We're done if the task set is deleted
				return nil
			}

			taskSet, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			if IsWorkflowCompleted(taskSet) {
				log.WithField("taskset", ae.WorkflowName).Info("stopped agent")
				os.Exit(0)
			}

			for nodeID, tmpl := range taskSet.Spec.Tasks {
				nodeID, tmpl := nodeID, tmpl
				go func() {
					// Do not work on completed tasks
					if _, ok := ae.CompletedTasks[nodeID]; ok {
						return
					}

					ae.KeyLock.Lock(nodeID)
					defer ae.KeyLock.Unlock(nodeID)

					log.WithFields(log.Fields{"nodeID": nodeID}).Error("SIMON Processing task")
					patch, err := ae.processTask(ctx, nodeID, tmpl)
					if err != nil {
						log.WithFields(log.Fields{"error": err, "nodeID": nodeID}).Error("Error in agent task")
					}
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

func (ae *AgentExecutor) processResult(ctx context.Context, nodeID string, patch []byte, taskSetInterface v1alpha1.WorkflowTaskSetInterface) {
	log.WithFields(log.Fields{"workflow": ae.WorkflowName, "namespace": ae.Namespace}).Infof("Patching content, %s", patch)

	obj, err := taskSetInterface.Patch(ctx, ae.WorkflowName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.WithError(err).WithField("taskset", obj).Errorf("failed to update the taskset")
	}

	log.WithField("taskset", obj).Infof("Patched content, %s", patch)

	ae.CompletedTasks[nodeID] = struct{}{}
}

func (ae *AgentExecutor) executeHTTPTemplate(ctx context.Context, tmpl wfv1.Template) (*wfv1.Outputs, error) {
	httpTemplate := tmpl.HTTP
	request, err := http.NewRequest(httpTemplate.Method, httpTemplate.URL, bytes.NewBufferString(httpTemplate.Body))
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
		request.Header.Add(header.Name, value)
	}
	response, err := argohttp.SendHttpRequest(request)
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
