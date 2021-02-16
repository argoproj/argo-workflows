package httptemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/antonmedv/expr"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/plugin"
)

type task struct {
	key string
	req *http.Request
}

type impl struct {
	work     *workqueue.Type
	outcomes *sync.Map
}

func New() plugin.TemplateExecutor {
	work := workqueue.NewNamed("http-requests")
	outcomes := &sync.Map{}
	return &impl{work, outcomes}
}

func (p *impl) Init(req plugin.InitReq, resp *plugin.InitResp) error {
	resp.PluginTemplateTypes = []string{"http"}
	return nil
}

func (p *impl) Run(req plugin.RunReq) {
	go func() {
		for func() bool {
			item, shutdown := p.work.Get()
			if shutdown {
				return false
			}
			t := item.(task)
			key := t.key
			if _, ok := p.outcomes.Load(key); ok {
				log.WithField("key", key).Info("HTTP request already done")
				return true
			}
			log.WithFields(log.Fields{"key": key, "method": t.req.Method, "url": t.req.URL}).Info("making HTTP request")
			body, err := executeHTTPRequest(t.req)
			parts := strings.Split(t.key, "/")
			defer req.Notify(parts[0], parts[1])
			if err != nil {
				log.WithField("key", key).WithError(err).Error("HTTP request failed")
				p.outcomes.Store(key, err)
			} else {
				log.WithField("key", key).Error("HTTP request successful")
				p.outcomes.Store(key, body)
			}
			return true
		}() {
		}
	}()

	<-req.Done

	p.work.ShutDown()
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

type httpReq struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

func (p *impl) ExecuteNode(req plugin.ExecuteNodeReq, resp *wfv1.NodeStatus) error {
	h := &httpReq{}
	if err := req.Template.Plugin.UnmarshalTo(h); err != nil {
		return err
	}

	r, err := http.NewRequest(h.Method, h.URL, bytes.NewBuffer(h.Body))
	if err != nil {
		return err
	}
	for k, v := range h.Headers {
		r.Header.Add(k, v)
	}

	p.work.Add(task{req.Workflow.Namespace + "/" + req.Workflow.Name + "/" + req.Node.Name, r})

	resp.Phase = wfv1.NodePending

	return nil
}

func (p *impl) ReconcileNode(req plugin.ReconcileNodeReq, resp *wfv1.NodeStatus) error {
	key := req.Workflow.Namespace + "/" + req.Workflow.Name + "/" + req.Node.Name
	value, exists := p.outcomes.LoadAndDelete(key)
	if !exists {
		return nil
	}
	switch v := value.(type) {
	case json.RawMessage:
		log.WithField("body", string(v)).Info("HTTP body")
		outputs, err := parseOutputs(req.Template.Outputs, v)
		if err != nil {
			return err
		}
		resp.Phase = wfv1.NodeSucceeded
		resp.Outputs = &outputs
		return nil
	case error:
		resp.Phase = wfv1.NodeFailed
		resp.Message = v.Error()
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
