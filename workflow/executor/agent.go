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

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
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
	CompleteTask      map[string]struct{}
	Plugins           []executorplugins.TemplateExecutor
}

type templateExecutor = func(ctx context.Context, tmpl wfv1.Template, reply *wfv1.NodeResult) (time.Duration, error)

func (ae *AgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	defer log.Info("stopped agent")

	log := log.WithField("workflow", ae.WorkflowName)

	taskSetInterface := ae.WorkflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(ae.Namespace)
	for {
		wfWatch, err := taskSetInterface.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + ae.WorkflowName})
		if err != nil {
			return err
		}

		for event := range wfWatch.ResultChan() {
			log.WithField("event", event.Type).Infof("Event")

			if event.Type == watch.Deleted {
				// We're done if the task set is deleted
				return nil
			}

			obj, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			if IsWorkflowCompleted(obj) {
				return nil
			}
			tasks := obj.Spec.Tasks
			for nodeID, tmpl := range tasks {

				if _, ok := ae.CompleteTask[nodeID]; ok {
					continue
				}

				var executeTemplate templateExecutor
				switch {
				case tmpl.HTTP != nil:
					executeTemplate = ae.executeHTTPTemplate
				case tmpl.Plugin != nil:
					executeTemplate = ae.executePluginTemplate
				default:
					return fmt.Errorf("plugins cannot execute: unknown task type: %v", tmpl.GetType())
				}

				result := wfv1.NodeResult{}
				log := log.WithField("nodeID", nodeID)
				if requeue, err := executeTemplate(ctx, tmpl, &result); err != nil {
					result.Phase = wfv1.NodeFailed
					result.Message = err.Error()
				} else if requeue > 0 {
					time.AfterFunc(requeue, func() {
						log.Info("re-queue")
						delete(ae.CompleteTask, nodeID)
					})
				}

				log.WithField("phase", result.Phase).
					WithField("message", result.Message).
					WithField("outputs", result.Outputs).
					Info("Node result")

				nodeResults := map[string]wfv1.NodeResult{}

				nodeResults[nodeID] = result

				patch, err := json.Marshal(map[string]interface{}{"status": wfv1.WorkflowTaskSetStatus{Nodes: nodeResults}})

				if err != nil {
					return errors.InternalWrapError(err)
				}

				_, err = taskSetInterface.Patch(ctx, ae.WorkflowName, types.MergePatchType, patch, metav1.PatchOptions{})
				if err != nil {
					log.WithError(err).Error("failed to update the taskset")
				}
				log.Info("updated taskset")

				if result.Fulfilled() {
					ae.CompleteTask[nodeID] = struct{}{}
				}
			}
		}
	}
}

func (ae *AgentExecutor) executeHTTPTemplate(ctx context.Context, tmpl wfv1.Template, reply *wfv1.NodeResult) (time.Duration, error) {
	httpTemplate := tmpl.HTTP
	request, err := http.NewRequest(httpTemplate.Method, httpTemplate.URL, bytes.NewBufferString(httpTemplate.Body))
	if err != nil {
		return 0, err
	}

	for _, header := range httpTemplate.Headers {
		value := header.Value
		if header.ValueFrom != nil && header.ValueFrom.SecretKeyRef != nil {
			secret, err := util.GetSecrets(ctx, ae.ClientSet, ae.Namespace, header.ValueFrom.SecretKeyRef.Name, header.ValueFrom.SecretKeyRef.Key)
			if err != nil {
				return 0, err
			}
			value = string(secret)
		}
		request.Header.Add(header.Name, value)
	}
	response, err := argohttp.SendHttpRequest(request, httpTemplate.TimeoutSeconds)
	if err != nil {
		return 0, err
	}
	reply.Phase = wfv1.NodeSucceeded
	reply.Outputs = &wfv1.Outputs{
		Parameters: []wfv1.Parameter{{Name: "result", Value: wfv1.AnyStringPtr(response)}},
	}
	return 0, nil
}

func (ae *AgentExecutor) executePluginTemplate(_ context.Context, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
	args := executorplugins.ExecuteTemplateArgs{
		Workflow: &executorplugins.Workflow{
			ObjectMeta: executorplugins.ObjectMeta{Name: ae.WorkflowName},
		},
		Template: &tmpl,
	}
	reply := &executorplugins.ExecuteTemplateReply{}
	for _, plug := range ae.Plugins {
		if err := plug.ExecuteTemplate(args, reply); err != nil {
			return 0, err
		} else if reply.Node != nil {
			*result = *reply.Node
			return reply.GetRequeue(), nil
		}
	}
	return 0, nil
}

func IsWorkflowCompleted(wts *wfv1.WorkflowTaskSet) bool {
	value := wts.Labels[common.LabelKeyCompleted]
	return value == "true"
}
