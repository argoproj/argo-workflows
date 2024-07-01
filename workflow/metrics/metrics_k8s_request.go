package metrics

import (
	"context"
	"net/http"
	"time"

	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/util/k8s"
)

const (
	nameK8sRequestTotal    = `k8s_request_total`
	nameK8sRequestDuration = `k8s_request_duration`
)

func addK8sRequests(_ context.Context, m *Metrics) error {
	err := m.createInstrument(int64Counter,
		nameK8sRequestTotal,
		"Number of kubernetes requests executed.",
		"{request}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.createInstrument(float64Histogram,
		nameK8sRequestDuration,
		"Duration of kubernetes requests executed.",
		"s",
		withDefaultBuckets([]float64{0.1, 0.2, 0.5, 1.0, 2.0, 5.0, 10.0, 20.0, 60.0, 180.0}),
		withAsBuiltIn(),
	)
	// Register this metrics with the global
	k8sMetrics.metrics = m
	return err
}

type metricsRoundTripper struct {
	ctx          context.Context
	roundTripper http.RoundTripper
	metrics      *Metrics
}

// This is a messy global as we need to register as a roundtripper before
// we can instantiate metrics
var k8sMetrics metricsRoundTripper

func (m metricsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	startTime := time.Now()
	x, err := m.roundTripper.RoundTrip(r)
	duration := time.Since(startTime)
	if x != nil && m.metrics != nil {
		verb, kind := k8s.ParseRequest(r)
		attribs := instAttribs{
			{name: labelRequestKind, value: kind},
			{name: labelRequestVerb, value: verb},
			{name: labelRequestCode, value: x.StatusCode},
		}
		(*m.metrics).addInt(m.ctx, nameK8sRequestTotal, 1, attribs)
		(*m.metrics).record(m.ctx, nameK8sRequestDuration, duration.Seconds(), attribs)
	}
	return x, err
}

func AddMetricsTransportWrapper(ctx context.Context, config *rest.Config) *rest.Config {
	wrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wrap != nil {
			rt = wrap(rt)
		}
		k8sMetrics.ctx = ctx
		k8sMetrics.roundTripper = rt
		return &k8sMetrics
	}
	return config
}
