package types

import (
	"k8s.io/client-go/kubernetes"

	eventsource "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"

	workflow "github.com/argoproj/argo/pkg/client/clientset/versioned"
)

type Clients struct {
	Workflow    workflow.Interface
	Sensor      sensor.Interface
	EventSource eventsource.Interface
	Kubernetes  kubernetes.Interface
}
