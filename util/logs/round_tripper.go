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
	x, err := l.roundTripper.RoundTrip(r)
	if x != nil {
		log.Debugf("%s %s %v", r.Method, r.URL, x.StatusCode)
	}
	return x, err
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
