package metrics

import (
	"context"
	"net/http"
	"time"

	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/util/k8s"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addK8sRequests(_ context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentK8sRequestTotal)
	if err != nil {
		return err
	}
	err = m.CreateBuiltinInstrument(telemetry.InstrumentK8sRequestDuration)
	// Register this metrics with the global
	k8sMetrics.metrics = m
	return err
}

type metricsRoundTripperContext struct {
	ctx     context.Context
	metrics *Metrics
}

type metricsRoundTripper struct {
	*metricsRoundTripperContext
	roundTripper http.RoundTripper
}

// This is a messy global as we need to register as a roundtripper before
// we can instantiate metrics
var k8sMetrics metricsRoundTripperContext

func (m metricsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	startTime := time.Now()
	x, err := m.roundTripper.RoundTrip(r)
	duration := time.Since(startTime)
	if x != nil && m.metrics != nil {
		verb, kind := k8s.ParseRequest(r)
		attribs := telemetry.InstAttribs{
			{Name: telemetry.AttribRequestKind, Value: kind},
			{Name: telemetry.AttribRequestVerb, Value: verb},
			{Name: telemetry.AttribRequestCode, Value: x.StatusCode},
		}
		(*m.metrics).AddInt(m.ctx, telemetry.InstrumentK8sRequestTotal.Name(), 1, attribs)
		(*m.metrics).Record(m.ctx, telemetry.InstrumentK8sRequestDuration.Name(), duration.Seconds(), attribs)
	}
	return x, err
}

func AddMetricsTransportWrapper(ctx context.Context, config *rest.Config) *rest.Config {
	wrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wrap != nil {
			rt = wrap(rt)
		}
		return &metricsRoundTripper{roundTripper: rt, metricsRoundTripperContext: &k8sMetrics}
	}
	return config
}
