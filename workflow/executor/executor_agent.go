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
	"k8s.io/apimachinery/pkg/watch"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (we *WorkflowExecutor) Agent(ctx context.Context) error {
	// return we.agentUsingWorkflowAgent(ctx)
	return we.agentUsingWorkflowNode(ctx)
}

// nolint:unused
func (we *WorkflowExecutor) agentUsingWorkflowAgent(ctx context.Context) error {
	i := we.workflowInterface.ArgoprojV1alpha1().WorkflowAgents(we.Namespace)
	for {
		w, err := i.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + we.workflowName})
		if err != nil {
			return err
		}
		for event := range w.ResultChan() {
			if event.Type == watch.Deleted {
				return nil // we're done when deleted: exit with code 0
			}
			x, ok := event.Object.(*wfv1.WorkflowAgent)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			for _, n := range x.Status.Nodes.Filter(func(n wfv1.NodeStatus) bool { return !n.Fulfilled() }) {
				tmpl := x.GetTemplateByName(n.TemplateName)
				switch n.Type {
				case wfv1.NodeTypeHTTP:
					result := wfv1.NodeStatus{}
					if err := we.executeHTTPTemplate(ctx, *tmpl); err != nil {
						result.Phase = wfv1.NodeFailed
						result.Message = err.Error()
					} else {
						result.Phase = wfv1.NodeSucceeded
					}
					data, err := json.Marshal(&wfv1.WorkflowAgent{Status: wfv1.WorkflowAgentStatus{Nodes: wfv1.Nodes{n.ID: result}}})
					if err != nil {
						return err
					}
					if _, err := i.Patch(ctx, we.workflowName, types.MergePatchType, data, metav1.PatchOptions{}); err != nil {
						return err
					}
				default:
					return fmt.Errorf("agent cannot execute %s", n.Type)
				}
			}
		}
	}
}

func (we *WorkflowExecutor) agentUsingWorkflowNode(ctx context.Context) error {
	i := we.workflowInterface.ArgoprojV1alpha1().WorkflowNodes(we.Namespace)
	for {
		w, err := i.Watch(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + we.workflowName})
		if err != nil {
			return err
		}
		for event := range w.ResultChan() {
			x, ok := event.Object.(*wfv1.WorkflowNode)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			if event.Type != watch.Added { // we only process add events because all updates we do ourselves
				continue
			}
			if x.Status.Fulfilled() {
				continue
			}
			tmpl := x.Spec
			switch tmpl.GetType() {
			case wfv1.TemplateTypeHTTP:
				result := wfv1.NodeStatus{}
				if err := we.executeHTTPTemplate(ctx, tmpl); err != nil {
					result.Phase = wfv1.NodeFailed
					result.Message = err.Error()
				} else {
					result.Phase = wfv1.NodeSucceeded
				}
				data, err := json.Marshal(&wfv1.WorkflowNode{Status: result})
				if err != nil {
					return err
				}
				if _, err := i.Patch(ctx, x.Name, types.MergePatchType, data, metav1.PatchOptions{}); err != nil {
					return err
				}
			default:
				return fmt.Errorf("agent cannot execute %s", tmpl.GetType())
			}
		}
	}
}

func (we *WorkflowExecutor) executeHTTPTemplate(ctx context.Context, tmpl wfv1.Template) error {
	h := tmpl.HTTP
	in, err := http.NewRequest(h.Method, h.URL, bytes.NewBuffer(h.Body))
	if err != nil {
		return err
	}
	for _, v := range h.Headers {
		value := v.Value
		if v.ValueFrom != nil || v.ValueFrom.SecretKeyRef != nil {
			secret, err := util.GetSecrets(ctx, we.ClientSet, we.Namespace, v.ValueFrom.SecretKeyRef.Name, v.ValueFrom.SecretKeyRef.Key)
			if err != nil {
				return err
			}
			value = string(secret)
		}
		in.Header.Add(v.Name, value)
	}
	log.WithField("url", in.URL).Info("making HTTP request")
	out, err := http.DefaultClient.Do(in)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{"url": in.URL, "status": out.Status}).Info("HTTP request made")
	if out.StatusCode >= 300 {
		return fmt.Errorf(out.Status)
	}
	return nil
}
