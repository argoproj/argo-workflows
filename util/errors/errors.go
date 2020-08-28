package errors

import (
	"net"
	"net/url"
	"strings"

	apierr "k8s.io/apimachinery/pkg/api/errors"

	argoerrs "github.com/argoproj/argo/errors"
)

func IsTransientErr(err error) bool {
	if err == nil {
		return false
	}
	err = argoerrs.Cause(err)
	return isExceededQuotaErr(err) || apierr.IsTooManyRequests(err) || isResourceQuotaConflictErr(err) || isTransientNetworkErr(err)
}

func isExceededQuotaErr(err error) bool {
	return apierr.IsForbidden(err) && strings.Contains(err.Error(), "exceeded quota")
}

func isResourceQuotaConflictErr(err error) bool {
	return apierr.IsConflict(err) && strings.Contains(err.Error(), "Operation cannot be fulfilled on resourcequota")
}

func isTransientNetworkErr(err error) bool {
	switch err.(type) {
	case net.Error:
		switch err.(type) {
		case *net.DNSError, *net.OpError, net.UnknownNetworkError:
			return true
		case *url.Error:
			// For a URL error, where it replies back "connection closed"
			// retry again.
			return strings.Contains(err.Error(), "Connection closed by foreign host")
		default:
			if strings.Contains(err.Error(), "net/http: TLS handshake timeout") {
				// If error is - tlsHandshakeTimeoutError, retry.
				return true
			} else if strings.Contains(err.Error(), "i/o timeout") {
				// If error is - tcp timeoutError, retry.
				return true
			} else if strings.Contains(err.Error(), "connection timed out") {
				// If err is a net.Dial timeout, retry.
				return true
			}
		}
	}
	return false
}
