package agent

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// swagger:parameters executeTemplate
type ExecuteTemplateRequest struct {
	// in: body
	Body ExecuteTemplateArgs
}

type ExecuteTemplateArgs struct {
	Workflow *wfv1.Workflow `json:"workflow"`
	Template *wfv1.Template `json:"template"`
}

// swagger:response executeTemplate
type ExecuteTemplateResponse struct {
	// in: body
	Body ExecuteTemplateReply
}

type ExecuteTemplateReply struct {
	Node *wfv1.NodeResult `json:"node,omitempty"`
}

type TemplateExecutor interface {
	// swagger:route POST /template.execute executeTemplate
	ExecuteTemplate(args ExecuteTemplateArgs, reply *ExecuteTemplateReply) error
}
