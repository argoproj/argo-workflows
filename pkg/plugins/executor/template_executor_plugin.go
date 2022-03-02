package executor

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// swagger:parameters executeTemplate
type ExecuteTemplateRequest struct {
	// in: body
	// Required: true
	Body ExecuteTemplateArgs
}

type ExecuteTemplateArgs struct {
	// Required: true
	Workflow *Workflow `json:"workflow"`
	// Required: true
	Template *wfv1.Template `json:"template"`
}

// swagger:response executeTemplate
type ExecuteTemplateResponse struct {
	// in: body
	Body ExecuteTemplateReply
}

type ExecuteTemplateReply struct {
	Node    *wfv1.NodeResult `json:"node,omitempty"`
	Requeue *metav1.Duration `json:"requeue,omitempty"`
}

func (r ExecuteTemplateReply) GetRequeue() time.Duration {
	if r.Requeue != nil {
		return r.Requeue.Duration
	}
	return 0
}

type TemplateExecutor interface {
	// swagger:route POST /template.execute executeTemplate
	//     Responses:
	//       200: executeTemplate
	ExecuteTemplate(ctx context.Context, args ExecuteTemplateArgs, reply *ExecuteTemplateReply) error
}
