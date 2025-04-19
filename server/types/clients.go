package types

import (
	events "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
)

type Clients struct {
	Dynamic    dynamic.Interface
	Workflow   workflow.Interface
	Events     events.Interface
	Kubernetes kubernetes.Interface
}
