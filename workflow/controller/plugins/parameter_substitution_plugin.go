package plugins

import wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

type ParameterPreSubstitutionArgs struct {
	Workflow *wfv1.Workflow
	Template *wfv1.Template
}

type ParameterPreSubstitutionReply struct {
	Parameters map[string]string
}

type ParameterSubstitutionPlugin interface {
	ParameterPreSubstitution(args ParameterPreSubstitutionArgs, reply *ParameterPreSubstitutionReply) error
}
