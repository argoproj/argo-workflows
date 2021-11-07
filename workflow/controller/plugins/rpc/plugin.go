package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins"
)

type plugin struct {
	address string
	invalid map[string]bool
}

func New(spec map[string]interface{}) (interface{}, error) { //nolint:deadcode,unparam
	return &plugin{address: spec["address"].(string), invalid: map[string]bool{}}, nil
}

func main() {
	// main funcs are never called in a Go plugin
}

func (p *plugin) call(method string, args interface{}, reply interface{}) error {
	if p.invalid[method] {
		return nil
	}
	req, err := json.Marshal(args)
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("%s/%s", p.address, method), "application/json", bytes.NewBuffer(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		return json.NewDecoder(resp.Body).Decode(reply)
	case 404:
		p.invalid[method] = true
		_, err := io.Copy(io.Discard, resp.Body)
		return err
	default:
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", string(data))
	}
}

var _ plugins.WorkflowLifecycleHook = &plugin{}

func (p *plugin) WorkflowPreOperate(args plugins.WorkflowPreOperateArgs, reply *plugins.WorkflowPreOperateReply) error {
	return p.call("WorkflowLifecycleHook.WorkflowPreOperate", args, reply)
}

func (p *plugin) WorkflowPreUpdate(args plugins.WorkflowPreUpdateArgs, reply *plugins.WorkflowPreUpdateReply) error {
	return p.call("WorkflowLifecycleHook.WorkflowPreUpdate", args, reply)
}

var _ plugins.NodeLifecycleHook = &plugin{}

func (p *plugin) NodePreExecute(args plugins.NodePreExecuteArgs, reply *plugins.NodePreExecuteReply) error {
	return p.call("NodeLifecycleHook.NodePreExecute", args, reply)
}

func (p *plugin) NodePostExecute(args plugins.NodePostExecuteArgs, reply *plugins.NodePostExecuteReply) error {
	return p.call("NodeLifecycleHook.NodePostExecute", args, reply)
}

var _ plugins.PodLifecycleHook = &plugin{}

func (p *plugin) PodPreCreate(args plugins.PodPreCreateArgs, reply *plugins.PodPreCreateReply) error {
	return p.call("PodLifecycleHook.PodPreCreate", args, reply)
}

func (p *plugin) PodPostCreate(args plugins.PodPostCreateArgs, reply *plugins.PodPostCreateReply) error {
	return p.call("PodLifecycleHook.PodPostCreate", args, reply)
}

var _ plugins.ParameterSubstitutionPlugin = &plugin{}

func (p *plugin) ParameterPreSubstitution(args plugins.ParameterPreSubstitutionArgs, reply *plugins.ParameterPreSubstitutionReply) error {
	return p.call("ParameterSubstitutionPlugin.ParameterPreSubstitution", args, reply)
}
