package controller

import wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

// swagger:parameters addParameters
type ParameterPreSubstitutionRequest struct {
	// in: body
	Body ParameterPreSubstitutionArgs
}

// swagger:response addParameters
type ParameterPreSubstitutionResponse struct {
	// in: body
	Body ParameterPreSubstitutionReply
}

type ParameterPreSubstitutionArgs struct {
	Workflow *wfv1.Workflow `json:"workflow"`
	Template *wfv1.Template `json:"template"`
}

type ParameterPreSubstitutionReply struct {
	Parameters map[string]string `json:"parameters,omitempty"`
}

type ParameterSubstitutionPlugin interface {
	// swagger:route POST /parameters.add addParameters
	//     Responses:
	//       200: addParameters
	AddParameters(args ParameterPreSubstitutionArgs, reply *ParameterPreSubstitutionReply) error
}
