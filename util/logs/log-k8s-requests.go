package logs

import (
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/util/k8s"
)

type k8sLogRoundTripper struct {
	roundTripper http.RoundTripper
}

func (m k8sLogRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	x, err := m.roundTripper.RoundTrip(r)
	if x != nil {
		statusCode := x.StatusCode
		var response string
		if responseBody := x.Body; (200 > statusCode || statusCode >= 300) && responseBody != nil {
			bytes, err := ioutil.ReadAll(responseBody)
			if err == nil {
				response = string(bytes)
			}
		}
		verb, kind := k8s.ParseRequest(r)
		log.Infof("%s %s %d %s", verb, kind, statusCode, response)
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
