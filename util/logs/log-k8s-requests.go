package logs

import (
	"context"
	"net/http"
	"time"

	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/util/k8s"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

var (
	extraLongThrottleLatency = 5 * time.Second
)

type k8sLogRoundTripper struct {
	roundTripper http.RoundTripper
	// nolint: containedctx
	ctx context.Context
}

func (m k8sLogRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	logger := logging.RequireLoggerFromContext(m.ctx)
	now := time.Now()
	x, err := m.roundTripper.RoundTrip(r)
	latency := time.Since(now)

	if x != nil {
		verb, kind := k8s.ParseRequest(r)
		if latency > extraLongThrottleLatency {
			logger.WithFields(logging.Fields{"latency": latency, "verb": verb, "url": r.URL.String()}).Warn(m.ctx, "Waited for K8S request")
		}
		logger.WithFields(logging.Fields{"verb": verb, "kind": kind, "status": x.StatusCode}).Debug(m.ctx, "K8S request")
	}
	return x, err
}

func AddK8SLogTransportWrapper(ctx context.Context, config *rest.Config) *rest.Config {
	wrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wrap != nil {
			rt = wrap(rt)
		}
		return &k8sLogRoundTripper{roundTripper: rt, ctx: ctx}
	}
	return config
}
