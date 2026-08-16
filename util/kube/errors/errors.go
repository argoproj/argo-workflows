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
	}
	return false
}
