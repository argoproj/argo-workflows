package apiclient

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
