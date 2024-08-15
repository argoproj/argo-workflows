package metrics

import (
	"context"
	"net/http"

	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/util/k8s"
)

const (
	nameK8sRequestTotal = `k8s_request_total`
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
	x, err := m.roundTripper.RoundTrip(r)
	if x != nil && m.metrics != nil {
		verb, kind := k8s.ParseRequest(r)
		attribs := instAttribs{
			{name: labelRequestKind, value: kind},
			{name: labelRequestVerb, value: verb},
			{name: labelRequestCode, value: x.StatusCode},
		}
		(*m.metrics).addInt(m.ctx, nameK8sRequestTotal, 1, attribs)
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
