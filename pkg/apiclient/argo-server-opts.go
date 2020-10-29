package apiclient

import "fmt"

type ArgoServerOpts struct {
	// argo server URL
	URL                              string
	Secure, InsecureSkipVerify, HTTP bool
}

func (o ArgoServerOpts) GetURL() string {
	if o.Secure {
		return "https://" + o.URL
	}
	return "http://" + o.URL
}

func (o ArgoServerOpts) String() string {
	return fmt.Sprintf("(url=%s,secure=%v,insecureSkipVerify=%v,http=%v)", o.URL, o.Secure, o.InsecureSkipVerify, o.HTTP)
}
