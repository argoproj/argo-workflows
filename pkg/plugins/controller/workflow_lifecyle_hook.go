package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type WorkflowPreOperateArgs struct {
	Workflow *wfv1.Workflow `json:"workflow"`
}

type WorkflowPreOperateReply struct {
	Workflow *wfv1.Workflow `json:"workflow,omitempty"`
}

type WorkflowPostOperateArgs struct {
	Old *wfv1.Workflow `json:"old"`
	New *wfv1.Workflow `json:"new"`
}

type WorkflowPostOperateReply struct {
}

type WorkflowLifecycleHook interface {
	// WorkflowPreOperate is called prior to reconciliation, allowing you to modify the workflow before execution.
	WorkflowPreOperate(args WorkflowPreOperateArgs, reply *WorkflowPreOperateReply) error
	// WorkflowPostOperate is called prior to persisting a changed workflow.
	WorkflowPostOperate(args WorkflowPostOperateArgs, reply *WorkflowPostOperateReply) error
}
