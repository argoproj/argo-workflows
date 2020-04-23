package apiclient

type ArgoServerOpts struct {
	// argo server URL
	URL                        string
	Secure, InsecureSkipVerify bool
}
