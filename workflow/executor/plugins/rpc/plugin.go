package rpc

import (
	plugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/agent"
	rpc "github.com/argoproj/argo-workflows/v3/workflow/util/plugins"
)

type plugin struct{ rpc.Plugin }

func New(address string) *plugin {
	return &plugin{Plugin: rpc.New(address)}
}

var _ plugins.TemplateExecutor = &plugin{}

func (p *plugin) ExecuteTemplate(args plugins.ExecuteTemplateArgs, reply *plugins.ExecuteTemplateReply) error {
	return p.Call("template.execute", args, reply)
}
