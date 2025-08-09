package apiclient

import (
	"fmt"
	"net/http"
)

type ArgoServerOpts struct {
	// argo server host port, must be `host:port`, e.g. localhost:2746
	URL string
	// root path prefix for API requests
	RootPath           string
	// base href for UI access
	BaseHRef           string
	Secure             bool
	InsecureSkipVerify bool
	// whether or not to use HTTP1
	HTTP1 bool
	// use custom http client
	HTTP1Client *http.Client
	Headers     []string
}

func (o ArgoServerOpts) GetURL() string {
	if o.Secure {
		return "https://" + o.URL + o.RootPath
	}
	return "http://" + o.URL + o.RootPath
}

func (o ArgoServerOpts) String() string {
	return fmt.Sprintf("(url=%s,rootPath=%s,baseHRef=%s,secure=%v,insecureSkipVerify=%v,http=%v)", o.URL, o.RootPath, o.BaseHRef, o.Secure, o.InsecureSkipVerify, o.HTTP1)
}
