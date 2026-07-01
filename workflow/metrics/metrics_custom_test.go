package metrics

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type dummyObserver struct {
	metric.Observer
}

func (d *dummyObserver) ObserveFloat64(metric.Float64Observable, float64, ...metric.ObserveOption) {}
func (d *dummyObserver) ObserveInt64(metric.Int64Observable, int64, ...metric.ObserveOption)       {}

func TestUpsertCustomMetric_Concurrency(t *testing.T) {
	ctx := context.Background()
	m, _, err := CreateDefaultTestMetrics()
	require.NoError(t, err)

	trueVal := true
	metricSpec := &wfv1.Prometheus{
		Name: "test_concurrency_metric",
		Help: "test help",
		Gauge: &wfv1.Gauge{
			Value:    "1",
			Realtime: &trueVal,
		},
	}

	numGoroutines := 20
	var wg sync.WaitGroup
	stop := make(chan struct{})

	// 1. Controller: Upsert metrics
	for i := range numGoroutines {
		owner := fmt.Sprintf("owner-%d", i)
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					_ = m.UpsertCustomMetric(ctx, metricSpec, owner, func() float64 { return 1.0 })
				}
			}
		})
	}

	// 2. Workflow: Complete metrics
	for i := range 5 {
		owner := fmt.Sprintf("owner-%d", i)
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					m.CompleteRealtimeMetricsForWfUID(owner)
				}
			}
		})
	}

	// 3. Prometheus: Scrape metrics
	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				base := m.GetInstrument(metricSpec.Name)
				if base != nil {
					inst := &customInstrument{Instrument: base}
					_ = inst.customCallback(ctx, &dummyObserver{})
				}
			}
		}
	})

	// 4. Background: GC
	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				m.runCustomGC(1 * time.Hour)
			}
		}
	})

	// Run stress test for a short duration
	time.Sleep(1 * time.Second)
	close(stop)
	wg.Wait()
}

func TestUpsertCustomMetric_PanicWindow(t *testing.T) {
	ctx := context.Background()
	m, _, err := CreateDefaultTestMetrics()
	require.NoError(t, err)

	metricSpec := &wfv1.Prometheus{
		Name: "test_panic_window_metric",
		Help: "test help",
		Gauge: &wfv1.Gauge{
			Value: "1",
		},
	}

	// Create the metric. This registers the callback.
	err = m.createCustomMetric(metricSpec)
	require.NoError(t, err)

	inst := m.GetInstrument(metricSpec.Name)
	require.NotNil(t, inst)

	// Simulate a scrape immediately.
	// This should not panic because we now set userdata before registration
	// and the callback handles nil userdata gracefully.
	customInst := &customInstrument{Instrument: inst}
	err = customInst.customCallback(ctx, &dummyObserver{})
	assert.NoError(t, err)
}
