package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// swagger:parameters workflowPreOperate
type WorkflowPreOperateRequest struct {
	// in: body
	Body WorkflowPreOperateArgs
}
type WorkflowPreOperateArgs struct {
	Workflow *wfv1.Workflow `json:"workflow"`
}

// swagger:response workflowPreOperate
type WorkflowPreOperateResponse struct {
	// in: body
	Body WorkflowPreOperateReply
}
type WorkflowPreOperateReply struct {
	Workflow *wfv1.Workflow `json:"workflow,omitempty"`
}

// swagger:parameters workflowPostOperate
type WorkflowPostOperateRequest struct {
	// in: body
	Body WorkflowPreOperateArgs
}

type WorkflowPostOperateArgs struct {
	Old *wfv1.Workflow `json:"old"`
	New *wfv1.Workflow `json:"new"`
}

// swagger:response workflowPostOperate
type WorkflowPostOperateResponse struct {
	// in: body
	Body WorkflowPreOperateReply
}

type WorkflowPostOperateReply struct {
}

type WorkflowLifecycleHook interface {
	// WorkflowPreOperate is called prior to reconciliation, allowing you to modify the workflow before execution.
	// swagger:route POST /workflow.preOperate workflowPreOperate
	//     Responses:
	//       200: workflowPreOperate
	WorkflowPreOperate(args WorkflowPreOperateArgs, reply *WorkflowPreOperateReply) error

	// WorkflowPostOperate is called prior to persisting a changed workflow.
	// swagger:route POST /workflow.postOperate workflowPostOperate
	//     Responses:
	//       200: workflowPostOperate
	WorkflowPostOperate(args WorkflowPostOperateArgs, reply *WorkflowPostOperateReply) error
}
