package rpc

import (
	"time"

	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	rpc "github.com/argoproj/argo-workflows/v3/workflow/util/plugins"
)

type plugin struct{ rpc.Plugin }

func New(address string) *plugin {
	return &plugin{Plugin: rpc.New(address, 30*time.Second)}
}

var _ executorplugins.TemplateExecutor = &plugin{}

func (p *plugin) ExecuteTemplate(args executorplugins.ExecuteTemplateArgs, reply *executorplugins.ExecuteTemplateReply) error {
	return p.Call("template.execute", args, reply)
}
