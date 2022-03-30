package types

import (
	"fmt"

	eventsource "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
)

type Profile = Clients

func NewProfile(restConfig *restclient.Config) (*Profile, error) {
	logs.AddK8SLogTransportWrapper(restConfig)
	metrics.AddMetricsTransportWrapper(restConfig)
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failure to create dynamic client: %w", err)
	}
	wfClient, err := wfclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failure to create workflow client: %w", err)
	}
	eventSourceClient, err := eventsource.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failure to create event source client: %w", err)
	}
	sensorClient, err := sensor.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failure to create sensor client: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failure to create kubernetes client: %w", err)
	}
	return &Profile{
		RESTConfig:  restConfig,
		Dynamic:     dynamicClient,
		Workflow:    wfClient,
		Sensor:      sensorClient,
		EventSource: eventSourceClient,
		Kubernetes:  kubeClient,
	}, nil
}
