package plugins

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type WorkflowPreOperateArgs struct{}

type WorkflowPreOperateReply struct {
	Workflow *wfv1.Workflow
}

type WorkflowPreUpdateArgs struct {
	Old *wfv1.Workflow
}

type WorkflowPreUpdateReply struct {
	New *wfv1.Workflow
}

type WorkflowLifecycleHook interface {
	// WorkflowPreOperate is called prior to reconciliation, allowing you to modify the workflow before execution.
	WorkflowPreOperate(args WorkflowPreOperateArgs, reply *WorkflowPreOperateReply) error
	// WorkflowPreUpdate is called prior to persisting a changed workflow.
	WorkflowPreUpdate(args WorkflowPreUpdateArgs, reply *WorkflowPreUpdateReply) error
}
