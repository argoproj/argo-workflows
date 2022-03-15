package apiclient

import (
	"fmt"
)

type ArgoServerOpts struct {
	// argo server host port, must be `host:port`, e.g. localhost:2746
	URL string
	// any base path needed (e.g. due to being behind an ingress)
	Path               string
	Secure             bool
	InsecureSkipVerify bool
	// whether or not to use HTTP1
	HTTP1   bool
	Headers []string
}

func (o ArgoServerOpts) GetURL() string {
	if o.Secure {
		return "https://" + o.URL + o.Path
	}
	return "http://" + o.URL + o.Path
}

func (o ArgoServerOpts) String() string {
	return fmt.Sprintf("(url=%s,path=%s,secure=%v,insecureSkipVerify=%v,http=%v)", o.URL, o.Path, o.Secure, o.InsecureSkipVerify, o.HTTP1)
}
