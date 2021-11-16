package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// swagger:parameters addParameters
type ParameterPreSubstitutionRequest struct {
	// in: body
	// Required: true
	Body ParameterPreSubstitutionArgs
}

// swagger:response addParameters
type ParameterPreSubstitutionResponse struct {
	// in: body
	// Required: true
	Body ParameterPreSubstitutionReply
}

type ParameterPreSubstitutionArgs struct {
	// Required: true
	Workflow *Workflow `json:"workflow"`
	// Required: true
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
