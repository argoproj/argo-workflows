package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// legacyWorkflowCollector collects metrics about all workflows in the cluster
type controllerCollector struct {
	informer cache.SharedIndexInformer
	gauges   map[wfv1.NodePhase]prometheus.Gauge
}

func newControllerCollector(informer cache.SharedIndexInformer) *controllerCollector {
	getOptsByPhase := func(phase wfv1.NodePhase) prometheus.GaugeOpts {
		return prometheus.GaugeOpts{
			Namespace:   argoNamespace,
			Subsystem:   workflowsSubsystem,
			Name:        "count",
			Help:        "Number of Workflows currently accessible by the controller by status NEW",
			ConstLabels: map[string]string{"status": string(phase)},
		}
	}

	var controllerCollector controllerCollector

	controllerCollector.informer = informer
	controllerCollector.addWorkflowInformerHandler()

	controllerCollector.gauges = map[wfv1.NodePhase]prometheus.Gauge{
		wfv1.NodePending:   prometheus.NewGauge(getOptsByPhase(wfv1.NodePending)),
		wfv1.NodeRunning:   prometheus.NewGauge(getOptsByPhase(wfv1.NodeRunning)),
		wfv1.NodeSucceeded: prometheus.NewGauge(getOptsByPhase(wfv1.NodeSucceeded)),
		wfv1.NodeSkipped:   prometheus.NewGauge(getOptsByPhase(wfv1.NodeSkipped)),
		wfv1.NodeFailed:    prometheus.NewGauge(getOptsByPhase(wfv1.NodeFailed)),
		wfv1.NodeError:     prometheus.NewGauge(getOptsByPhase(wfv1.NodeError)),
	}

	return &controllerCollector
}

func (cc *controllerCollector) addWorkflowInformerHandler() {
	cc.informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if gauge, ok := cc.gauges[getWfPhase(obj)]; ok {
					gauge.Inc()
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if gauge, ok := cc.gauges[getWfPhase(old)]; ok {
					gauge.Dec()
				}
				if gauge, ok := cc.gauges[getWfPhase(new)]; ok {
					gauge.Inc()
				}
			},
			DeleteFunc: func(obj interface{}) {
				if gauge, ok := cc.gauges[getWfPhase(obj)]; ok {
					gauge.Dec()
				}
			},
		},
	)
}

func getWfPhase(obj interface{}) wfv1.NodePhase {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return ""
	}
	phase, hasPhase, err := unstructured.NestedString(un.Object, "status", "phase")
	if err != nil || !hasPhase {
		return ""
	}

	return wfv1.NodePhase(phase)
}

// Describe implements the prometheus.Collector interface
func (wc *controllerCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range wc.gauges {
		ch <- metric.Desc()
	}
}

// Collect implements the prometheus.Collector interface
func (wc *controllerCollector) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range wc.gauges {
		ch <- metric
	}
}
