package metrics

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/metric"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

type RealTimeValueFunc func() float64

type customMetricValue struct {
	// We are faking a settable and incrementable up/down gauge/coutner
	// for compatibility with prometheus. callback() reads this.
	// This is used for counters, gauges and realtime custom metrics,
	// using either the prometheusValue or the
	// rtValueFunc in the realtime case.
	prometheusValue float64
	rtValueFunc     RealTimeValueFunc
	lastUpdated     time.Time
	labels          []*wfv1.MetricLabel
	key             string
}

type realtimeTracker struct {
	inst *telemetry.Instrument
	key  string
}

func (cmv *customMetricValue) getLabels() telemetry.InstAttribs {
	labels := make(telemetry.InstAttribs, len(cmv.labels))
	for i := range cmv.labels {
		labels[i] = telemetry.InstAttrib{Name: cmv.labels[i].Key, Value: cmv.labels[i].Value}
	}
	return labels
}

func customUserdata(i *telemetry.Instrument, requireSuccess bool) map[string]*customMetricValue {
	switch val := i.GetUserdata().(type) {
	case map[string]*customMetricValue:
		return val
	default:
		if requireSuccess {
			panic(fmt.Errorf("internal error: unexpected userdata on custom metric %s", i.GetName()))
		}
		return make(map[string]*customMetricValue)
	}
}

func getOrCreateValue(i *telemetry.Instrument, key string, labels []*wfv1.MetricLabel) *customMetricValue {
	if value, ok := customUserdata(i, true)[key]; ok {
		return value
	}
	newValue := customMetricValue{
		key:    key,
		labels: labels,
	}
	customUserdata(i, true)[key] = &newValue
	return &newValue
}

type customInstrument struct {
	*telemetry.Instrument
}

// Common callback for realtime and gauges
// For realtime this acts as a thunk to the calling convention
// For non-realtime we have to fake observability as prometheus provides
// up/down and set on the same gauge type, which otel forbids.
func (i *customInstrument) customCallback(_ context.Context, o metric.Observer) error {
	for _, value := range customUserdata(i.Instrument, true) {
		if value.rtValueFunc != nil {
			i.ObserveFloat(o, value.rtValueFunc(), value.getLabels())
		} else {
			i.ObserveFloat(o, value.prometheusValue, value.getLabels())
		}
	}
	return nil
}

// func addCustomMetrics(_ context.Context, m *Metrics) error {
// 	m.customMetrics = make(map[string]*customMetric, 0)
// 	return nil
// }

// GetCustomMetric returns a custom (or any) metric from it's key
// This is exported for legacy testing only
func (m *Metrics) GetCustomMetric(key string) *telemetry.Instrument {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	// It's okay to return nil metrics in this function
	return m.AllInstruments[key]
}

// CustomMetricExists returns if metric exists from its key
// This is exported for testing only
func (m *Metrics) CustomMetricExists(key string) bool {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	// It's okay to return nil metrics in this function
	return m.AllInstruments[key] != nil
}

// TODO labels on custom metrics
func (m *Metrics) matchExistingMetric(metricSpec *wfv1.Prometheus) (*telemetry.Instrument, error) {
	key := metricSpec.Name
	if inst, ok := m.AllInstruments[key]; ok {
		if inst.GetDescription() != metricSpec.Help {
			return nil, fmt.Errorf("Help for metric %s is already set to %s, it cannot be changed", metricSpec.Name, inst.GetDescription())
		}
		wantedType := metricSpec.GetMetricType()
		switch inst.GetOtel().(type) {
		case *metric.Float64ObservableGauge:
			if wantedType != wfv1.MetricTypeGauge && !metricSpec.IsRealtime() {
				return nil, fmt.Errorf("Found existing gauge for custom metric %s of type %s", metricSpec.Name, wantedType)
			}
		case *metric.Float64ObservableUpDownCounter:
			if wantedType != wfv1.MetricTypeCounter {
				return nil, fmt.Errorf("Found existing counter for custom metric %s of type %s", metricSpec.Name, wantedType)
			}
		case *metric.Float64Histogram:
			if wantedType != wfv1.MetricTypeHistogram {
				return nil, fmt.Errorf("Found existing histogram for custom metric %s of type %s", metricSpec.Name, wantedType)
			}
		default:
			return nil, fmt.Errorf("Found unwanted type %s for custom metric %s of type %s", reflect.TypeOf(inst.GetOtel()), metricSpec.Name, wantedType)
		}
		return inst, nil
	}
	return nil, nil
}

func (m *Metrics) ensureBaseMetric(metricSpec *wfv1.Prometheus, ownerKey string) (*telemetry.Instrument, error) {
	metric, err := m.matchExistingMetric(metricSpec)
	if err != nil {
		return nil, err
	}
	if metric != nil {
		m.attachCustomMetricToWorkflow(metricSpec, ownerKey)
		return metric, nil
	}
	err = m.createCustomMetric(metricSpec)
	if err != nil {
		return nil, err
	}
	m.attachCustomMetricToWorkflow(metricSpec, ownerKey)
	inst := m.AllInstruments[metricSpec.Name]
	if inst == nil {
		return nil, fmt.Errorf("Failed to create new metric %s", metricSpec.Name)
	}
	inst.SetUserdata(make(map[string]*customMetricValue))
	return inst, nil
}

func (m *Metrics) UpsertCustomMetric(ctx context.Context, metricSpec *wfv1.Prometheus, ownerKey string, valueFunc RealTimeValueFunc) error {
	if !IsValidMetricName(metricSpec.Name) {
		return fmt.Errorf("%s", invalidMetricNameError)
	}
	baseMetric, err := m.ensureBaseMetric(metricSpec, ownerKey)
	if err != nil {
		return err
	}
	metricValue := getOrCreateValue(baseMetric, metricSpec.GetKey(), metricSpec.Labels)
	metricValue.lastUpdated = time.Now()

	metricType := metricSpec.GetMetricType()
	switch {
	case metricSpec.IsRealtime():
		metricValue.rtValueFunc = valueFunc

	case metricType == wfv1.MetricTypeGauge:
		val, err := strconv.ParseFloat(metricSpec.Gauge.Value, 64)
		if err != nil {
			return err
		}
		switch metricSpec.Gauge.Operation {
		case wfv1.GaugeOperationAdd:
			metricValue.prometheusValue += val
		case wfv1.GaugeOperationSub:
			metricValue.prometheusValue -= val
		case wfv1.GaugeOperationSet:
			fallthrough
		default:
			metricValue.prometheusValue = val
		}
	case metricType == wfv1.MetricTypeHistogram:
		val, err := strconv.ParseFloat(metricSpec.Histogram.Value, 64)
		if err != nil {
			return err
		}
		baseMetric.Record(ctx, val, metricValue.getLabels())
	case metricType == wfv1.MetricTypeCounter:
		val, err := strconv.ParseFloat(metricSpec.Counter.Value, 64)
		if err != nil {
			return err
		}
		metricValue.prometheusValue += val
	default:
		return fmt.Errorf("invalid custom metric type")
	}
	return nil
}

func (m *Metrics) attachCustomMetricToWorkflow(metricSpec *wfv1.Prometheus, ownerKey string) {
	if metricSpec.IsRealtime() {
		// Must move to run each workflowkey
		for key := range m.realtimeWorkflows {
			if key == ownerKey {
				return
			}
		}
		m.realtimeWorkflows[ownerKey] = append(m.realtimeWorkflows[ownerKey], realtimeTracker{
			inst: m.AllInstruments[metricSpec.Name],
			key:  metricSpec.GetKey(),
		})
	}
}

func (m *Metrics) createCustomMetric(metricSpec *wfv1.Prometheus) error {
	metricType := metricSpec.GetMetricType()
	switch {
	case metricSpec.IsRealtime():
		err := m.createCustomGauge(metricSpec)
		if err != nil {
			return err
		}
		return nil
	case metricType == wfv1.MetricTypeGauge:
		return m.createCustomGauge(metricSpec)
	case metricType == wfv1.MetricTypeHistogram:
		return m.CreateInstrument(telemetry.Float64Histogram, metricSpec.Name, metricSpec.Help, "{item}", telemetry.WithDefaultBuckets(metricSpec.Histogram.GetBuckets()))
	case metricType == wfv1.MetricTypeCounter:
		err := m.CreateInstrument(telemetry.Float64ObservableUpDownCounter, metricSpec.Name, metricSpec.Help, "{item}")
		if err != nil {
			return err
		}
		inst := m.AllInstruments[metricSpec.Name]
		customInst := customInstrument{Instrument: inst}
		return inst.RegisterCallback(m.Metrics, customInst.customCallback)
	default:
		return fmt.Errorf("invalid metric spec")
	}
}

func (m *Metrics) createCustomGauge(metricSpec *wfv1.Prometheus) error {
	err := m.CreateInstrument(telemetry.Float64ObservableGauge, metricSpec.Name, metricSpec.Help, "{item}")
	if err != nil {
		return err
	}
	inst := m.AllInstruments[metricSpec.Name]
	customInst := customInstrument{Instrument: inst}
	return inst.RegisterCallback(m.Metrics, customInst.customCallback)
}

func (m *Metrics) runCustomGC(ttl time.Duration) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	for _, baseMetric := range m.AllInstruments {
		custom := customUserdata(baseMetric, false)
		for key, value := range custom {
			if time.Since(value.lastUpdated) > ttl {
				delete(custom, key)
			}
		}
	}
}

func (m *Metrics) customMetricsGC(ctx context.Context, ttl time.Duration) {
	if ttl == 0 {
		return
	}

	ticker := time.NewTicker(ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.runCustomGC(ttl)
		}
	}
}

func (m *Metrics) StopRealtimeMetricsForWfUID(key string) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if _, exists := m.realtimeWorkflows[key]; !exists {
		return
	}

	realtimeMetrics := m.realtimeWorkflows[key]
	for _, metric := range realtimeMetrics {
		delete(customUserdata(metric.inst, true), metric.key)
	}

	delete(m.realtimeWorkflows, key)
}
