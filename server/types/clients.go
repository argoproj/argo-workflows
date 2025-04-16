package types

import (
	eventsource "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	sensor "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
)

type Clients struct {
	Dynamic     dynamic.Interface
	Workflow    workflow.Interface
	Sensor      sensor.Interface
	EventSource eventsource.Interface
	Kubernetes  kubernetes.Interface
}
