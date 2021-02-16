package jsonrpc

import (
	"net/rpc/jsonrpc"
	"strings"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/plugin"
)

func (p *impl) cantFindMethodErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "rpc: can't find method")
}

func (p *impl) call(method string, req interface{}, resp interface{}) error {
	if p.unsupported[method] {
		return nil
	}
	client, err := jsonrpc.Dial("tcp", p.addr)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	log.Infof("calling %q", method)
	err = client.Call(method, req, resp)
	log.Infof("returned: %v, %v", resp, err)
	if p.cantFindMethodErr(err) {
		p.unsupported[method] = true
		return nil
	}
	return err
}

func New(addr string) plugin.TemplateExecutor {
	return &impl{addr, make(map[string]bool)}
}

type impl struct {
	addr        string
	unsupported map[string]bool
}

func (p *impl) Init(req plugin.InitReq, resp *plugin.InitResp) error {
	return p.call("init", req, resp)
}

func (p *impl) ExecuteNode(req plugin.ExecuteNodeReq, resp *wfv1.NodeStatus) error {
	return p.call("executeNode", req, resp)
}

func (p *impl) ReconcileNode(req plugin.ReconcileNodeReq, resp *wfv1.NodeStatus) error {
	return p.call("reconcileNode", req, resp)
}
