package metrics

import (
	"sync"
	"sync/atomic"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestUpsertCustomMetric_Concurrency(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	m, _, err := CreateDefaultTestMetrics(ctx)
	if err != nil {
		t.Fatal(err)
	}

	metricSpec := &wfv1.Prometheus{
		Name: "test_concurrency_metric",
		Help: "test help",
		Gauge: &wfv1.Gauge{
			Value: "1",
		},
	}

	numGoroutines := 200
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	start := make(chan struct{})
	var panicCount int32

	for range numGoroutines {
		go func() {
			defer wg.Done()
			<-start
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt32(&panicCount, 1)
				}
			}()
			_ = m.UpsertCustomMetric(ctx, metricSpec, "owner", func() float64 { return 1.0 })
		}()
	}

	close(start)
	wg.Wait()

	if panicCount > 0 {
		t.Errorf("Caught %d panics due to concurrency race condition", panicCount)
	}
}
