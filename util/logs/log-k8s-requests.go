package logs

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/util/k8s"
)

var (
	extraLongThrottleLatency = 5 * time.Second
)

type k8sLogRoundTripper struct {
	roundTripper http.RoundTripper
}

func (m k8sLogRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	now := time.Now()
	x, err := m.roundTripper.RoundTrip(r)
	latency := time.Since(now)

	if x != nil {
		verb, kind := k8s.ParseRequest(r)
		if latency > extraLongThrottleLatency {
			log.Warnf("Waited for %v, request: %s:%s", latency, verb, r.URL.String())
		}
		log.Debugf("%s %s %d", verb, kind, x.StatusCode)
	}
	return x, err
}

func AddK8SLogTransportWrapper(config *rest.Config) *rest.Config {
	wrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wrap != nil {
			rt = wrap(rt)
		}
		return &k8sLogRoundTripper{roundTripper: rt}
	}
	return config
}
