package plugins

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// Plugin functions must be RPC compatible: https://pkg.go.dev/net/rpc

type WorkflowPreOperateArgs struct {
	Workflow *wfv1.Workflow
}

type WorkflowPreOperateReply struct{}

type WorkflowPreUpdateArgs struct {
	Old *wfv1.Workflow
	New *wfv1.Workflow
}

type WorkflowPreUpdateReply struct{}

type WorkflowLifecycleHook interface {
	// WorkflowPreOperate is called prior to reconciliation, allowing you to modify the workflow before execution.
	WorkflowPreOperate(args WorkflowPreOperateArgs, reply *WorkflowPreOperateReply) error
	// WorkflowPreUpdate is called prior to persisting a changed workflow.
	WorkflowPreUpdate(args WorkflowPreUpdateArgs, reply *WorkflowPreUpdateReply) error
}

type ExecuteTemplateArgs struct {
	Workflow *wfv1.Workflow
	Template *wfv1.Template
}
type ExecuteTemplateReply struct {
	Node *wfv1.NodeStatus
}

type TemplateExecutor interface {
	// ExecuteTemplate is called when executing a template. It will called multiple times.
	// If the returned status is fulfilled, then the template will not itself be run.
	ExecuteTemplate(args ExecuteTemplateArgs, reply *ExecuteTemplateReply) error
}
