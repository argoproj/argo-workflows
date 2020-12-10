package logs

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

type loggingRoundTripper struct {
	roundTripper http.RoundTripper
}

func (l *loggingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	log.Debugf("%s %s", r.Method, r.URL)
	return l.roundTripper.RoundTrip(r)
}

func AddLoggingTransportWrapper(config *rest.Config) *rest.Config {
	wrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wrap != nil {
			rt = wrap(rt)
		}
		return &loggingRoundTripper{roundTripper: rt}
	}
	return config
}
