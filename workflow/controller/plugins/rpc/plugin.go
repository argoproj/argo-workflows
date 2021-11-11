package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"

	plugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
	"github.com/argoproj/argo-workflows/v3/util/errors"
)

type plugin struct {
	address string
	invalid map[string]bool
}

func New(data map[string]string) (interface{}, error) { //nolint:deadcode,unparam
	address, ok := data["address"]
	if !ok {
		return nil, fmt.Errorf("address not specfied")
	}
	return &plugin{address: address, invalid: map[string]bool{}}, nil
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
		log.WithField("address", p.invalid).
			WithField("method", method).
			Info("method not found, never calling again")
		p.invalid[method] = true
		_, err := io.Copy(io.Discard, resp.Body)
		return err
	case 503:
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.NewErrTransient(string(data))
	default:
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s: %s", resp.Status, string(data))
	}
}

var _ plugins.WorkflowLifecycleHook = &plugin{}

func (p *plugin) WorkflowPreOperate(args plugins.WorkflowPreOperateArgs, reply *plugins.WorkflowPreOperateReply) error {
	return p.call("WorkflowLifecycleHook.WorkflowPreOperate", args, reply)
}

func (p *plugin) WorkflowPostOperate(args plugins.WorkflowPostOperateArgs, reply *plugins.WorkflowPostOperateReply) error {
	return p.call("WorkflowLifecycleHook.WorkflowPostOperate", args, reply)
}

var _ plugins.NodeLifecycleHook = &plugin{}

func (p *plugin) NodePreExecute(args plugins.NodePreExecuteArgs, reply *plugins.NodePreExecuteReply) error {
	return p.call("NodeLifecycleHook.NodePreExecute", args, reply)
}

func (p *plugin) NodePostExecute(args plugins.NodePostExecuteArgs, reply *plugins.NodePostExecuteReply) error {
	return p.call("NodeLifecycleHook.NodePostExecute", args, reply)
}

var _ plugins.ParameterSubstitutionPlugin = &plugin{}

func (p *plugin) ParameterPreSubstitution(args plugins.ParameterPreSubstitutionArgs, reply *plugins.ParameterPreSubstitutionReply) error {
	return p.call("ParameterSubstitutionPlugin.ParameterPreSubstitution", args, reply)
}
