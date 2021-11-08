package controller

import wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

type ParameterPreSubstitutionArgs struct {
	Workflow *wfv1.Workflow `json:"workflow"`
	Template *wfv1.Template `json:"template"`
}

type ParameterPreSubstitutionReply struct {
	Parameters map[string]string `json:"parameters,omitempty"`
}

type ParameterSubstitutionPlugin interface {
	ParameterPreSubstitution(args ParameterPreSubstitutionArgs, reply *ParameterPreSubstitutionReply) error
}
