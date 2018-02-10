package retry

import (
	"net"
	"net/url"
	"strings"
	"time"

	argoerrs "github.com/argoproj/argo/errors"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// DefaultRetry is a default retry backoff settings when retrying API calls
var DefaultRetry = wait.Backoff{
	Steps:    5,
	Duration: 10 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

// IsRetryableKubeAPIError returns if the error is a retryable kubernetes error
func IsRetryableKubeAPIError(err error) bool {
	// get original error if it was wrapped
	err = argoerrs.Cause(err)
	if apierr.IsNotFound(err) || apierr.IsForbidden(err) || apierr.IsInvalid(err) || apierr.IsMethodNotSupported(err) {
		return false
	}
	return true
}

// IsRetryableNetworkError returns whether or not the error is a retryable network error
func IsRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	// get original error if it was wrapped
	err = argoerrs.Cause(err)
	errStr := err.Error()

	switch err.(type) {
	case net.Error:
		switch err.(type) {
		case *net.DNSError, *net.OpError, net.UnknownNetworkError:
			return true
		case *url.Error:
			// For a URL error, where it replies back "connection closed"
			// retry again.
			if strings.Contains(errStr, "Connection closed by foreign host") {
				return true
			}
		default:
			if strings.Contains(errStr, "net/http: TLS handshake timeout") {
				// If error is - tlsHandshakeTimeoutError, retry.
				return true
			} else if strings.Contains(errStr, "i/o timeout") {
				// If error is - tcp timeoutError, retry.
				return true
			} else if strings.Contains(errStr, "connection timed out") {
				// If err is a net.Dial timeout, retry.
				return true
			}
		}
	}
	return false
}
