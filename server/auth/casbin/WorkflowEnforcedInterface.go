package casbin

import (
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"k8s.io/client-go/discovery"
)

type WorkflowEnforcedInterface struct {
	delegate workflow.Interface
}

func (c WorkflowEnforcedInterface) Discovery() discovery.DiscoveryInterface {
	panic("Discovery not supported")
}

func (c WorkflowEnforcedInterface) ArgoprojV1alpha1() v1alpha1.ArgoprojV1alpha1Interface {
	return c
}

func WrapWorkflowInterface(c workflow.Interface) workflow.Interface {
	return &WorkflowEnforcedInterface{c}
}
