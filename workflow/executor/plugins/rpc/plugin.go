package rpc

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	rpc "github.com/argoproj/argo-workflows/v3/workflow/util/plugins"
)

type plugin struct{ rpc.Plugin }

func New(address string) *plugin {
	return &plugin{Plugin: rpc.New(address, 30*time.Second, wait.Backoff{
		Duration: time.Second,
		Jitter:   0.2,
		Factor:   2,
		Steps:    5,
	})}
}

var _ executorplugins.TemplateExecutor = &plugin{}

func (p *plugin) ExecuteTemplate(args executorplugins.ExecuteTemplateArgs, reply *executorplugins.ExecuteTemplateReply) error {
	return p.Call("template.execute", args, reply)
}
