package executor

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type ExecuteTemplateArgs struct {
	Workflow *wfv1.Workflow   `json:"workflow"`
	Template *wfv1.Template   `json:"template"`
	Node     *wfv1.NodeStatus `json:"node"`
}

type ExecuteTemplateReply struct {
	Node *wfv1.NodeStatus `json:"node,omitempty"`
}

type TemplateExecutor interface {
	ExecuteTemplate(args ExecuteTemplateArgs, reply *ExecuteTemplateReply) error
}
