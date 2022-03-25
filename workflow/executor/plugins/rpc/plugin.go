package rpc

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	rpc "github.com/argoproj/argo-workflows/v3/workflow/util/plugins"
)

type plugin struct{ rpc.Client }

func New(address, token string) *plugin {
	return &plugin{Client: rpc.New(address, token, 30*time.Second, wait.Backoff{
		Duration: time.Second,
		Jitter:   0.2,
		Factor:   2,
		Steps:    5,
	})}
}

func (p *plugin) ExecuteTemplate(ctx context.Context, args executorplugins.ExecuteTemplateArgs, reply *executorplugins.ExecuteTemplateReply) error {
	return p.Call(ctx, "template.execute", args, reply)
}
