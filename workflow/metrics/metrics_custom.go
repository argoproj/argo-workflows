package metrics

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
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
	completed       bool
}

type customMetricUserData struct {
	mutex  sync.RWMutex
	values map[string]*customMetricValue
}

func newUserData() *customMetricUserData {
	return &customMetricUserData{
		values: make(map[string]*customMetricValue),
	}
}

func (ud *customMetricUserData) GetValue(key string) *customMetricValue {
	ud.mutex.RLock()
	defer ud.mutex.RUnlock()
	val, ok := ud.values[key]
	if !ok {
		return nil
	}
	return val
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

func customUserData(i *telemetry.Instrument, requireSuccess bool) *customMetricUserData {
	switch val := i.GetUserdata().(type) {
	case *customMetricUserData:
		return val
	default:
		if requireSuccess {
			panic(fmt.Errorf("internal error: unexpected userdata on custom metric %s", i.GetName()))
		}
		return nil
	}
}

func getOrCreateValue(i *telemetry.Instrument, key string, labels []*wfv1.MetricLabel) *customMetricValue {
	ud := customUserData(i, true)
	ud.mutex.Lock()
	defer ud.mutex.Unlock()
	if value, ok := ud.values[key]; ok {
		return value
	}
	newValue := customMetricValue{
		key:    key,
		labels: labels,
	}
	ud.values[key] = &newValue
	return &newValue
}

type customInstrument struct {
	*telemetry.Instrument
}

// Common callback for realtime and gauges
// For realtime this acts as a thunk to the calling convention
// For non-realtime we have to fake observability as prometheus provides
// up/down and set on the same gauge type, which otel forbids.
func (i *customInstrument) customCallback(ctx context.Context, o metric.Observer) error {
	ud := customUserData(i.Instrument, true)
	ud.mutex.RLock()
	defer ud.mutex.RUnlock()
	for _, value := range ud.values {
		if value.rtValueFunc != nil {
			i.ObserveFloat(ctx, o, value.rtValueFunc(), value.getLabels())
		} else {
			i.ObserveFloat(ctx, o, value.prometheusValue, value.getLabels())
		}
	}
	return nil
}

// GetCustomMetric returns a custom (or any) metric from it's key
// This is exported for legacy testing only
func (m *Metrics) GetCustomMetric(key string) *telemetry.Instrument {
	// It's okay to return nil metrics in this function
	return m.GetInstrument(key)
}

// CustomMetricExists returns if metric exists from its key
// This is exported for testing only
func (m *Metrics) CustomMetricExists(key string) bool {
	return m.GetCustomMetric(key) != nil
}

// TODO labels on custom metrics
func (m *Metrics) matchExistingMetric(metricSpec *wfv1.Prometheus) (*telemetry.Instrument, error) {
	key := metricSpec.Name
	if inst := m.GetInstrument(key); inst != nil {
		if inst.GetDescription() != metricSpec.Help {
			return nil, fmt.Errorf("help for metric %s is already set to %s, it cannot be changed", metricSpec.Name, inst.GetDescription())
		}
		wantedType := metricSpec.GetMetricType()
		switch inst.GetOtel().(type) {
		case *metric.Float64ObservableGauge:
			if wantedType != wfv1.MetricTypeGauge && !metricSpec.IsRealtime() {
				return nil, fmt.Errorf("found existing gauge for custom metric %s of type %s", metricSpec.Name, wantedType)
			}
		case *metric.Float64ObservableCounter:
			if wantedType != wfv1.MetricTypeCounter {
				return nil, fmt.Errorf("found existing counter for custom metric %s of type %s", metricSpec.Name, wantedType)
			}
		case *metric.Float64Histogram:
			if wantedType != wfv1.MetricTypeHistogram {
				return nil, fmt.Errorf("found existing histogram for custom metric %s of type %s", metricSpec.Name, wantedType)
			}
		default:
			return nil, fmt.Errorf("found unwanted type %s for custom metric %s of type %s", reflect.TypeOf(inst.GetOtel()), metricSpec.Name, wantedType)
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
	inst := m.GetInstrument(metricSpec.Name)
	if inst == nil {
		return nil, fmt.Errorf("failed to create new metric %s", metricSpec.Name)
	}
	inst.SetUserdata(newUserData())
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
		m.realtimeMutex.Lock()
		defer m.realtimeMutex.Unlock()
		// Must move to run each workflowkey
		for key := range m.realtimeWorkflows {
			if key == ownerKey {
				return
			}
		}
		m.realtimeWorkflows[ownerKey] = append(m.realtimeWorkflows[ownerKey], realtimeTracker{
			inst: m.GetInstrument(metricSpec.Name),
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
		err := m.CreateInstrument(telemetry.Float64ObservableCounter, metricSpec.Name, metricSpec.Help, "{item}")
		if err != nil {
			return err
		}
		inst := m.GetInstrument(metricSpec.Name)
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
	inst := m.GetInstrument(metricSpec.Name)
	customInst := customInstrument{Instrument: inst}
	return inst.RegisterCallback(m.Metrics, customInst.customCallback)
}

func (m *Metrics) runCustomGC(ttl time.Duration) {
	m.IterateROInstruments(func(baseMetric *telemetry.Instrument) {
		ud := customUserData(baseMetric, false)
		if ud == nil {
			return
		}
		ud.mutex.Lock()
		for key, value := range ud.values {
			if time.Since(value.lastUpdated) > ttl {
				switch {
				case value.rtValueFunc != nil && value.completed:
					delete(ud.values, key)
				case value.rtValueFunc == nil:
					delete(ud.values, key)
				}
			}
		}
		ud.mutex.Unlock()
	})
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

type operation int

const (
	Complete operation = iota
	Delete
)

func (m *Metrics) handleRealtimeMetricsForWfUID(key string, op operation) {
	m.realtimeMutex.Lock()
	defer m.realtimeMutex.Unlock()
	if _, exists := m.realtimeWorkflows[key]; !exists {
		return
	}
	realtimeMetrics := m.realtimeWorkflows[key]
	for _, metric := range realtimeMetrics {
		ud := customUserData(metric.inst, true)
		ud.mutex.Lock()
		switch op {
		case Complete:
			for _, value := range ud.values {
				value.completed = true
			}
		case Delete:
			delete(ud.values, metric.key)
		}
		ud.mutex.Unlock()
	}
	if op == Delete {
		delete(m.realtimeWorkflows, key)
	}
}

func (m *Metrics) CompleteRealtimeMetricsForWfUID(key string) {
	m.handleRealtimeMetricsForWfUID(key, Complete)
}

func (m *Metrics) DeleteRealtimeMetricsForWfUID(key string) {
	m.handleRealtimeMetricsForWfUID(key, Delete)
}
