package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/uuid"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/plugin"
)

func New(url string) plugin.TemplateExecutor {
	return &impl{url, make(map[string]bool)}
}

type impl struct {
	url         string
	unsupported map[string]bool
}

func (p *impl) Init(req plugin.InitReq, resp *plugin.InitResp) error {
	return p.call("init", req, resp)
}

func (p *impl) ExecuteNode(req plugin.ExecuteNodeReq, resp *wfv1.NodeStatus) error {
	return p.call("executeNode", req, resp)
}

type clientRequest struct {
	Method string         `json:"method"`
	Params [1]interface{} `json:"params"`
	Id     string         `json:"id"`
}

type clientResponse struct {
	Id     string           `json:"id"`
	Result *json.RawMessage `json:"result"`
	Error  interface{}      `json:"error"`
}

func (p *impl) call(method string, req interface{}, resp interface{}) error {
	if p.unsupported[method] {
		return nil
	}

	body := &bytes.Buffer{}

	if err := json.NewEncoder(body).Encode(clientRequest{Method: method, Params: [1]interface{}{req}, Id: string(uuid.NewUUID())}); err != nil {
		return fmt.Errorf("failed to encode %q request: %w", method, err)
	}
	log.Infof("calling %q", method)
	post, err := http.Post(p.url, "application/json", body)
	if err != nil {
		return fmt.Errorf("failed to call %q: %w", method, err)
	}
	defer func() { _ = post.Body.Close() }()

	switch post.StatusCode {
	case http.StatusOK:
		v := &clientResponse{}
		if err := json.NewDecoder(post.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode %q response: %w", method, err)
		}
		if v.Error != nil {
			return fmt.Errorf("call %q returned error: %v", method, v.Error)
		}
		return json.Unmarshal(*v.Result, resp)
	case http.StatusNotImplemented:
		p.unsupported[method] = true
		return nil
	default:
		return fmt.Errorf(post.Status)
	}
}
