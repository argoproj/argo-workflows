package metrics

import (
	"context"
	"sync"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"

	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	*telemetry.Metrics

	callbacks         Callbacks
	realtimeMutex     sync.Mutex
	realtimeWorkflows map[string][]realtimeTracker
}

func New(ctx context.Context, serviceName, prometheusName string, config *telemetry.Config, callbacks Callbacks, extraOpts ...metricsdk.Option) (*Metrics, error) {
	m, err := telemetry.NewMetrics(ctx, serviceName, prometheusName, config, extraOpts...)
	if err != nil {
		return nil, err
	}

	err = m.Populate(ctx,
		telemetry.AddVersion,
		telemetry.AddDeprecationCounter,
	)
	if err != nil {
		return nil, err
	}

	metrics := &Metrics{
		Metrics:           m,
		callbacks:         callbacks,
		realtimeWorkflows: make(map[string][]realtimeTracker),
	}

	err = metrics.populate(ctx,
		addIsLeader,
		addPodPhaseGauge,
		addPodPhaseCounter,
		addPodMissingCounter,
		addPodPendingCounter,
		addWorkflowPhaseGauge,
		addCronWfTriggerCounter,
		addCronWfPolicyCounter,
		addWorkflowPhaseCounter,
		addWorkflowTemplateCounter,
		addWorkflowTemplateHistogram,
		addOperationDurationHistogram,
		addErrorCounter,
		addLogCounter,
		addK8sRequests,
		addWorkflowConditionGauge,
		addWorkQueueMetrics,
	)
	if err != nil {
		return nil, err
	}

	go metrics.customMetricsGC(ctx, config.TTL)

	return metrics, nil
}

type addMetric func(context.Context, *Metrics) error

func (m *Metrics) populate(ctx context.Context, adders ...addMetric) error {
	for _, adder := range adders {
		if err := adder(ctx, m); err != nil {
			return err
		}
	}
	return nil
}
