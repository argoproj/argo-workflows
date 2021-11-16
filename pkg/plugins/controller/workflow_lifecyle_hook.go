package controller

// swagger:parameters workflowPreOperate
type WorkflowPreOperateRequest struct {
	// in: body
	// Required: true
	Body WorkflowPreOperateArgs
}
type WorkflowPreOperateArgs struct {
	// Required: true
	Workflow *Workflow `json:"workflow"`
}

// swagger:response workflowPreOperate
type WorkflowPreOperateResponse struct {
	// in: body
	// Required: true
	Body WorkflowPreOperateReply
}

type WorkflowPreOperateReply struct {
	Workflow *Workflow `json:"workflow,omitempty"`
}

// swagger:parameters workflowPostOperate
type WorkflowPostOperateRequest struct {
	// in: body
	// Required: true
	Body WorkflowPreOperateArgs
}

type WorkflowPostOperateArgs struct {
	// Required: true
	Old *Workflow `json:"old"`
	// Required: true
	New *Workflow `json:"new"`
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
