package kubeconfig

import "net/http"

type userAgentRoundTripper struct {
	agent string
	rt    http.RoundTripper
}

func (rt userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", rt.agent)
	return rt.rt.RoundTrip(req)
}

func NewUserAgentRoundTripper(agent string, rt http.RoundTripper) http.RoundTripper {
	return &userAgentRoundTripper{agent, rt}
}
