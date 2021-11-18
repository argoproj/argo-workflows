package rpc

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	controllerplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
	plugins "github.com/argoproj/argo-workflows/v3/workflow/util/plugins"
)

type plugin struct{ plugins.Plugin }

func New(address string) *plugin {
	return &plugin{Plugin: plugins.New(address, time.Second, wait.Backoff{Steps: 1})}
}

var _ controllerplugins.WorkflowLifecycleHook = &plugin{}

func (p *plugin) WorkflowPreOperate(args controllerplugins.WorkflowPreOperateArgs, reply *controllerplugins.WorkflowPreOperateReply) error {
	return p.Call("workflow.preOperate", args, reply)
}

func (p *plugin) WorkflowPostOperate(args controllerplugins.WorkflowPostOperateArgs, reply *controllerplugins.WorkflowPostOperateReply) error {
	return p.Call("workflow.postOperate", args, reply)
}

var _ controllerplugins.NodeLifecycleHook = &plugin{}

func (p *plugin) NodePreExecute(args controllerplugins.NodePreExecuteArgs, reply *controllerplugins.NodePreExecuteReply) error {
	return p.Call("node.preExecute", args, reply)
}

func (p *plugin) NodePostExecute(args controllerplugins.NodePostExecuteArgs, reply *controllerplugins.NodePostExecuteReply) error {
	return p.Call("node.postExecute", args, reply)
}

var _ controllerplugins.ParameterSubstitutionPlugin = &plugin{}

func (p *plugin) AddParameters(args controllerplugins.ParameterPreSubstitutionArgs, reply *controllerplugins.ParameterPreSubstitutionReply) error {
	return p.Call("parameters.add", args, reply)
}
