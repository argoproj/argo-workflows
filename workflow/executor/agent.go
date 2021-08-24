package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	argohttp "github.com/argoproj/argo-workflows/v3/workflow/executor/http"
)

type AgentExecutor struct {
	WorkflowName             string
	ClientSet                kubernetes.Interface
	WorkflowInterface        workflow.Interface
	WorkflowTaskSetInterface v1alpha1.WorkflowTaskSetInterface
	Namespace                string
	CompleteTask             map[string]struct{}
	Clients                  map[string]kubernetes.Interface
}

var keys = make(map[string]bool)

func (ae *AgentExecutor) Agent(ctx context.Context) error {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	watchPods := func(workflowTaskSet *wfv1.WorkflowTaskSet, clusterName, namespace string) {
		go func() {
			defer runtimeutil.HandleCrash()
			ae.watchPods(ctx, clusterName, namespace)
		}()
	}
	defer func() {
		defer runtimeutil.HandleCrash()
		ae.collectGarbage(ctx)
	}()

	for {
		wfWatch, err := ae.WorkflowTaskSetInterface.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + ae.WorkflowName})
		if err != nil {
			return err
		}

		for event := range wfWatch.ResultChan() {
			log.WithField("event", event.Type).Info("got event")

			if event.Type == watch.Deleted {
				// We're done if the task set is deleted
				return nil
			}

			obj, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			if IsWorkflowCompleted(obj) {
				log.Info("stopped agent")
				return nil
			}
			log.WithField("tasks", len(obj.Spec.Tasks)).Info("executing tasks")
			for nodeID, task := range obj.Spec.Tasks {
				_, completed := ae.CompleteTask[nodeID]
				log.WithField("completed", completed).
					WithField("nodeID", nodeID).
					WithField("task", wfv1.MustMarshallJSON(task)).
					Info("task")
				if completed {
					continue
				}
				result := wfv1.NodeResult{}
				switch {
				case task.HTTP != nil:
					if outputs, err := ae.executeHTTPTemplate(ctx, task.Template); err != nil {
						result.Phase = wfv1.NodeFailed
						result.Message = err.Error()
					} else {
						result.Phase = wfv1.NodeSucceeded
						result.Outputs = outputs
					}
				case task.Pod != nil:
					podName := task.Pod.Name
					clusterName := task.ClusterName
					namespace := task.Namespace
					log.WithField("clusterName", clusterName).
						WithField("namespace", namespace).
						WithField("name", podName).
						Info("creating workflow pod")
					_, err := ae.Clients[clusterName].CoreV1().
						Pods(namespace).
						Create(ctx, task.Pod, metav1.CreateOptions{})
					if err != nil {
						result.Phase = wfv1.NodeFailed
						result.Message = err.Error()
					} else {
						result.Phase = wfv1.NodePending
						result.Message = fmt.Sprintf("started pod %q on cluster %q namespace %q", podName, clusterName, namespace)
						watchPods(obj, clusterName, namespace)
					}
				default:
					return fmt.Errorf("agent cannot execute: unknown task type %q", task.GetType())
				}
				if err := ae.updateNodeResult(ctx, nodeID, result); err != nil {
					return err
				}
			}
		}
	}
}

func (ae *AgentExecutor) watchPods(ctx context.Context, clusterName string, namespace string ) {
	key := clusterName + "/" + namespace
	if keys[key] {
		return
	}
	keys[key] = true
	err := func() error {
		defer func() { keys[key] = false }()
		log.WithField("clusterName", clusterName).
			WithField("namespace", namespace).
			Info("watching workflow pods")
		w, err := ae.Clients[clusterName].CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + ae.WorkflowName})
		if err != nil {
			return err
		}
		defer w.Stop()
		for event := range w.ResultChan() {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			// challenge - we need to run assessNodeStatus
			node := wfv1.NodeResult{
				Phase:   wfv1.NodePhase(pod.Status.Phase),
				Message: pod.Status.Message,
			}
			if node.Phase.Fulfilled() {
				if outputStr, ok := pod.Annotations[common.AnnotationKeyOutputs]; ok {
					if err := json.Unmarshal([]byte(outputStr), node.Outputs); err != nil {
						node.Phase = wfv1.NodeError
						node.Message = err.Error()
					}
				}
			}
			if err := ae.updateNodeResult(ctx, pod.Name, node); err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		log.WithError(err).Error("failed to watch pods")
	}
}

func (ae *AgentExecutor) collectGarbage(ctx context.Context) {
	log.Info("garbage collecting (i.e. deleting) workflow pods")
	for key := range keys {
		parts := strings.Split(key, "/")
		clusterName := parts[0]
		namespace := parts[1]
		log.WithField("clusterName", clusterName).
			WithField("namespace", namespace).
			Info("deleting workflow pods")
		err := ae.Clients[clusterName].CoreV1().
			Pods(namespace).
			DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + ae.WorkflowName+",!workflows.argoproj.io/agent"})
		if err != nil {
			log.WithError(err).WithField("clusterName", clusterName).WithField("namespace", namespace).Error("failed to delete pods")
		}
	}
	data := wfv1.MustMarshallJSON(map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": nil,
		},
	})
	log.Info("removing finalizer")
	_, err := ae.ClientSet.CoreV1().Pods(ae.Namespace).Patch(ctx, os.Getenv(common.EnvVarPodName), types.MergePatchType, []byte(data), metav1.PatchOptions{})
	if err != nil {
		log.WithError(err).Error("failed to remove finalizer")
	}
}

func (ae *AgentExecutor) updateNodeResult(ctx context.Context, nodeID string, result wfv1.NodeResult) error {
	patch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"tasks": map[string]interface{}{
				nodeID: nil,
			},
		},
		"status": wfv1.WorkflowTaskSetStatus{
			Nodes: map[string]wfv1.NodeResult{
				nodeID: result,
			},
		},
	})
	if err != nil {
		return err
	}
	log.WithField("patch", string(patch)).Info("patching taskset")

	_, err = ae.WorkflowTaskSetInterface.Patch(ctx, ae.WorkflowName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	log.Info("updated taskset")

	ae.CompleteTask[nodeID] = struct{}{}

	return nil
}

func (ae *AgentExecutor) executeHTTPTemplate(ctx context.Context, tmpl wfv1.Template) (*wfv1.Outputs, error) {
	httpTemplate := tmpl.HTTP
	request, err := http.NewRequest(httpTemplate.Method, httpTemplate.URL, bytes.NewBuffer(httpTemplate.Body))
	if err != nil {
		return nil, err
	}

	for _, header := range httpTemplate.Headers {
		value := header.Value
		if header.ValueFrom != nil || header.ValueFrom.SecretKeyRef != nil {
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
	value := wts.Labels[common.LabelKeyCompleted]
	return value == "true"
}
