package errors

import (
	"errors"
	"net/http"
	"strings"

	apierr "k8s.io/apimachinery/pkg/api/errors"
)

// IsRequestEntityTooLargeErr determines if err is an error which indicates the size of the request
// was too large for the server to handle.
func IsRequestEntityTooLargeErr(err error) bool {
	var apiStatus apierr.APIStatus
	if errors.As(err, &apiStatus) {
		if apiStatus.Status().Code == http.StatusRequestEntityTooLarge {
			return true
		}
		// This also manifest with a 500 error with the message:
		// etcdserver: request is too large
		if strings.Contains(apiStatus.Status().Message, "request is too large") {
			return true
		}
		// Above 2MiB the API server's gRPC client to etcd refuses to send it:
		// rpc error: code = ResourceExhausted desc = trying to send message larger than max (2163613 vs. 2097152)
		if strings.Contains(apiStatus.Status().Message, "trying to send message larger than max") {
			return true
		}
	}
	return false
}
