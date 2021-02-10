package httptemplate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/antonmedv/expr"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/types"
)

type task struct {
	key string
	req *http.Request
}

var work = workqueue.NewNamed("http-requests")
var outcomes = &sync.Map{}
var kubeclientset kubernetes.Interface

func Init(k kubernetes.Interface) {
	kubeclientset = k
}

func Run(ctx context.Context, notify func(namespace, workflowName string), numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go func() {
			for runHTTPWorker(notify) {
			}
		}()
	}

	<-ctx.Done()

	work.ShutDown()
}

func runHTTPWorker(notify func(namespace string, workflowName string)) bool {
	item, shutdown := work.Get()
	if shutdown {
		return false
	}
	t := item.(task)
	key := t.key
	if _, ok := outcomes.Load(key); ok {
		log.WithField("key", key).Info("HTTP request already executed")
		return true
	}
	log.WithFields(log.Fields{"key": key, "method": t.req.Method, "url": t.req.URL}).Info("executing HTTP request")
	body, err := executeHTTPRequest(t.req)
	parts := strings.Split(t.key, "/")
	defer notify(parts[0], parts[1])
	if err != nil {
		log.WithField("key", key).WithError(err).Error("HTTP request failed")
		outcomes.Store(key, err)
	} else {
		log.WithField("key", key).Error("HTTP request successful")
		outcomes.Store(key, body)
	}
	return true
}

func executeHTTPRequest(req *http.Request) (json.RawMessage, error) {
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Body.Close() }()

	if r.StatusCode != 200 {
		return nil, fmt.Errorf("failed to execute HTTP request: %s", r.Status)
	}
	body := json.RawMessage{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to un-marshal HTTP body: %w", err)
	}
	return body, nil
}

func ExecuteNode(ctx context.Context, wf wfv1.Workflow, nodeName string, tmpl *wfv1.Template, initNode types.InitNodeFunc) (*wfv1.NodeStatus, error) {

	if node := wf.GetNodeByName(nodeName); node != nil {
		return node, nil
	}

	h := tmpl.HTTP
	r, err := http.NewRequest(h.Method, h.URL, bytes.NewBuffer(h.Body))
	if err != nil {
		return nil, err // TODO should this fail the node or error the workflow?
	}
	for _, v := range h.Headers {
		value := v.Value
		if v.ValueFrom != nil || v.ValueFrom.SecretKeyRef != nil {
			secret, err := util.GetSecrets(ctx, kubeclientset, wf.Namespace, v.ValueFrom.SecretKeyRef.Name, v.ValueFrom.SecretKeyRef.Key)
			if err != nil {
				return nil, err // TODO should this fail the node or error the workflow?
			}
			value = string(secret)
		}
		r.Header.Add(v.Name, value)
	}

	work.Add(task{wf.Namespace + "/" + wf.Name + "/" + nodeName, r})

	return initNode(wfv1.NodeTypeHTTP, wfv1.NodePending), nil
}

func ReconcileNode(wf wfv1.Workflow, node *wfv1.NodeStatus) error {
	key := wf.Namespace + "/" + wf.Name + "/" + node.Name
	value, exists := outcomes.LoadAndDelete(key)
	if !exists {
		return nil
	}
	tmpl := wf.GetTemplateByName(node.TemplateName)
	switch v := value.(type) {
	case json.RawMessage:
		log.WithField("body", v).Info("HTTP body")
		outputs, err := parseOutputs(tmpl.Outputs, v)
		if err != nil {
			return err
		}
		node.Phase = wfv1.NodeSucceeded
		node.Outputs = &outputs
		return nil
	case error:
		node.Phase = wfv1.NodeFailed
		node.Message = v.Error()
		return nil
	default:
		return fmt.Errorf("unexpected outcome")
	}
}

// this lengthy function shows the challenges around parsing output
func parseOutputs(outputs wfv1.Outputs, v json.RawMessage) (wfv1.Outputs, error) {
	log.WithField("outputs", outputs).Debug("template outputs")
	env := make(map[string]interface{})
	if err := json.Unmarshal(v, &env); err != nil {
		return outputs, fmt.Errorf("failed to un-marshall env: %w", err)
	}
	log.WithField("env", env).Debug("HTTP output expression environment")
	for i, p := range outputs.Parameters {
		if p.Value != nil {
			eval, err := expr.Eval(p.Value.String(), env)
			if err != nil {
				return outputs, fmt.Errorf("invalid output parameter expression %q: %w", p.Name, err)
			}
			p.Value = wfv1.AnyStringPtr(eval)
			outputs.Parameters[i] = p
		} else {
			return outputs, fmt.Errorf("invalid output parameter %q: must specify value", p.Name)
		}
	}
	return outputs, nil
}
