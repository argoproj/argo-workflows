package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/antonmedv/expr"
	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	exprenv "github.com/argoproj/argo-workflows/v3/util/expr/env"
)

func (we *WorkflowExecutor) Agent(ctx context.Context) error {
	i := we.workflowInterface.ArgoprojV1alpha1().WorkflowTaskSets(we.Namespace)
	for {
		w, err := i.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + we.workflowName})
		if err != nil {
			return err
		}
		for event := range w.ResultChan() {
			if event.Type == watch.Deleted {
				return nil // we're done when deleted: exit with code 0
			}
			x, ok := event.Object.(*wfv1.WorkflowTaskSet)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			for _, n := range x.Spec.Nodes {
				if x.Status != nil && x.Status.Nodes != nil && x.Status.Nodes[n.ID].Fulfilled() {
					continue
				}
				tmpl := x.GetTemplateByName(n.TemplateName)
				if tmpl == nil {
					return fmt.Errorf("tmpl nil")
				}
				switch n.Type {
				case wfv1.NodeTypeHTTP:
					result := wfv1.NodeResult{}
					if outputs, err := we.executeHTTPTemplate(ctx, *tmpl); err != nil {
						result.Phase = wfv1.NodeFailed
						result.Message = err.Error()
					} else {
						result.Phase = wfv1.NodeSucceeded
						result.Outputs = outputs
					}
					if x.Status == nil {
						x.Status = &wfv1.WorkflowTaskSetStatus{}
					}
					if x.Status.Nodes == nil {
						x.Status.Nodes = map[string]wfv1.NodeResult{}
					}
					x.Status.Nodes[n.ID] = result
					// con: we cannot patch status sub-resource, we must update the whole thing
					// could result in race-condition errors
					if _, err := i.UpdateStatus(ctx, x, metav1.UpdateOptions{}); err != nil {
						return err
					}
				default:
					return fmt.Errorf("agent cannot execute %s", n.Type)
				}
			}
		}
	}
}

func (we *WorkflowExecutor) executeHTTPTemplate(ctx context.Context, tmpl wfv1.Template) (*wfv1.Outputs, error) {
	h := tmpl.HTTP
	in, err := http.NewRequest(h.Method, h.URL, bytes.NewBuffer(h.Body))
	if err != nil {
		return nil, err
	}
	for _, v := range h.Headers {
		value := v.Value
		if v.ValueFrom != nil || v.ValueFrom.SecretKeyRef != nil {
			secret, err := util.GetSecrets(ctx, we.ClientSet, we.Namespace, v.ValueFrom.SecretKeyRef.Name, v.ValueFrom.SecretKeyRef.Key)
			if err != nil {
				return nil, err
			}
			value = string(secret)
		}
		in.Header.Add(v.Name, value)
	}
	log.WithField("url", in.URL).Info("making HTTP request")
	out, err := http.DefaultClient.Do(in)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"url": in.URL, "status": out.Status}).Info("HTTP request made")
	if out.StatusCode >= 300 {
		return nil, fmt.Errorf(out.Status)
	}

	data, err := ioutil.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}
	body := make(map[string]interface{})
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}

	o := &wfv1.Outputs{}
	for _, p := range tmpl.Outputs.Parameters {
		if p.Value != nil {
			result, err := expr.Eval(p.Value.String(), exprenv.GetFuncMap(map[string]interface{}{"body": body}))
			if err != nil {
				return nil, err
			}
			o.Parameters = append(o.Parameters, wfv1.Parameter{Name: p.Name, Value: wfv1.AnyStringPtr(result)})
		}
	}

	return o, nil
}
