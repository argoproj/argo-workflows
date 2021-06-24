package executor

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	argowait "github.com/argoproj/argo-workflows/v3/util/wait"
)

func (we *WorkflowExecutor) Agent(ctx context.Context) error {
	taskSetInterface := we.workflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(we.Namespace)
	for {
		wfWatch, err := taskSetInterface.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + we.workflowName})
		if err != nil {
			return err
		}
		log.Infof("watching")

		for event := range wfWatch.ResultChan() {
			log.Infof("watching, %v", event)
			if event.Type == watch.Deleted {
				// We're done if the task set is deleted
				return nil
			}

			obj, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			tasks := obj.Spec.Tasks.DeepCopy()
			for _, task := range tasks {
				if len(obj.Status.Nodes) > 0 && obj.Status.Nodes[task.NodeID].Fulfilled() {
					continue
				}
				switch {
				case task.Template.HTTP != nil:
					result := wfv1.NodeResult{}
					if outputs, err := we.executeHTTPTemplate(ctx, task.Template); err != nil {
						result.Phase = wfv1.NodeFailed
						result.Message = err.Error()
					} else {
						result.Phase = wfv1.NodeSucceeded
						result.Outputs = outputs
					}

					if obj.Status.Nodes == nil {
						obj.Status.Nodes = map[string]wfv1.NodeResult{}
					}
					obj.Status.Nodes[task.NodeID] = result
					err = argowait.Backoff(retry.DefaultBackoff, func() (bool, error) {
						obj, err = taskSetInterface.UpdateStatus(ctx, obj, metav1.UpdateOptions{})

						if errorsutil.IsTransientErr(err) || apierr.IsConflict(err) {
							return false, err
						}

						log.WithField("taskset", obj).Info("got back task set")
						return true, err
					})
					if err != nil {
						log.WithError(err).WithField("taskset", obj).Errorf("failed to update the taskset")
					}
				default:
					return fmt.Errorf("agent cannot execute: unknown task type")
				}
			}
		}
	}
}

func (we *WorkflowExecutor) executeHTTPTemplate(ctx context.Context, tmpl wfv1.Template) (*wfv1.Outputs, error) {
	httpTemplate := tmpl.HTTP
	request, err := http.NewRequest(httpTemplate.Method, httpTemplate.URL, bytes.NewBuffer(httpTemplate.Body))
	if err != nil {
		return nil, err
	}

	for _, header := range httpTemplate.Headers {
		value := header.Value
		if header.ValueFrom != nil || header.ValueFrom.SecretKeyRef != nil {
			secret, err := util.GetSecrets(ctx, we.ClientSet, we.Namespace, header.ValueFrom.SecretKeyRef.Name, header.ValueFrom.SecretKeyRef.Key)
			if err != nil {
				return nil, err
			}
			value = string(secret)
		}
		request.Header.Add(header.Name, value)
	}

	out, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{"url": request.URL, "status": out.Status}).Info("HTTP request made")
	if out.StatusCode >= 300 {
		return nil, fmt.Errorf(out.Status)
	}

	data, err := ioutil.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}

	outputs := &wfv1.Outputs{}
	outputs.Parameters = append(outputs.Parameters, wfv1.Parameter{Name: "result", Value: wfv1.AnyStringPtr(string(data))})

	return outputs, nil
}
