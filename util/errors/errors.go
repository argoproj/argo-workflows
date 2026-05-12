package errors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	apierr "k8s.io/apimachinery/pkg/api/errors"

	argoerrs "github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// IsTransientErr reports whether the error is transient and logs it.
func IsTransientErr(ctx context.Context, err error) bool {
	isTransient := IsTransientErrQuiet(ctx, err)
	if err != nil && !isTransient {
		logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "Non-transient error")
	}
	return isTransient
}

// IsTransientErrQuiet reports whether the error is transient and logs only if it is.
func IsTransientErrQuiet(ctx context.Context, err error) bool {
	isTransient := isTransientErr(err)
	if isTransient {
		logging.RequireLoggerFromContext(ctx).WithError(err).Info(ctx, "Transient error")
	}
	return isTransient
}

// isTransientErr reports whether the error is transient.
func isTransientErr(err error) bool {
	if err == nil {
		return false
	}
	err = argoerrs.Cause(err)
	return isExceededQuotaErr(err) ||
		apierr.IsTooManyRequests(err) ||
		isResourceQuotaConflictErr(err) ||
		isResourceQuotaTimeoutErr(err) ||
		isTransientNetworkErr(err) ||
		apierr.IsServerTimeout(err) ||
		apierr.IsTimeout(err) ||
		apierr.IsServiceUnavailable(err) ||
		isTransientEtcdErr(err) ||
		isTransientPodRejectedErr(err) ||
		matchTransientErrPattern(err) ||
		errors.Is(err, NewErrTransient("")) ||
		isTransientSqbErr(err)
}

func matchTransientErrPattern(err error) bool {
	// TRANSIENT_ERROR_PATTERN allows to specify the pattern to match for errors that can be seen as transient
	// and retryable.
	pattern, _ := os.LookupEnv("TRANSIENT_ERROR_PATTERN")
	if pattern == "" {
		return false
	}
	match, _ := regexp.MatchString(pattern, generateErrorString(err))
	return match
}

func isExceededQuotaErr(err error) bool {
	return apierr.IsForbidden(err) && strings.Contains(err.Error(), "exceeded quota")
}

func isResourceQuotaConflictErr(err error) bool {
	return apierr.IsConflict(err) && strings.Contains(err.Error(), "Operation cannot be fulfilled on resourcequota")
}

func isResourceQuotaTimeoutErr(err error) bool {
	return apierr.IsInternalError(err) && strings.Contains(err.Error(), "resource quota evaluation timed out")
}

func isTransientEtcdErr(err error) bool {
	// Some clusters expose these (transient) etcd errors to the caller
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "etcdserver: leader changed"):
		return true
	case strings.Contains(errStr, "etcdserver: request timed out"):
		return true
	case strings.Contains(errStr, "etcdserver: too many requests"):
		return true
	default:
		return false
	}
}

func isTransientPodRejectedErr(err error) bool {
	// This type of eviction happens before Pod could ever start
	return strings.Contains(err.Error(), "Pod was rejected:")
}

func isTransientNetworkErr(err error) bool {
	var dnsErr *net.DNSError
	var opErr *net.OpError
	var unknownNetErr net.UnknownNetworkError
	if errors.As(err, &dnsErr) || errors.As(err, &opErr) || errors.As(err, &unknownNetErr) {
		return true
	}

	errorString := generateErrorString(err)
	switch {
	case strings.Contains(errorString, "Connection closed by foreign host"):
		// For a URL error, where it replies back "connection closed"
		// retry again.
		return true
	case strings.Contains(errorString, "net/http: TLS handshake timeout"):
		// If error is - tlsHandshakeTimeoutError, retry.
		return true
	case strings.Contains(errorString, "i/o timeout"):
		// If error is - tcp timeoutError, retry.
		return true
	case strings.Contains(errorString, "connection timed out"):
		// If err is a net.Dial timeout, retry.
		return true
	case strings.Contains(errorString, "connection reset by peer"):
		// If err is a ECONNRESET, retry.
		return true
	case strings.Contains(errorString, "http2: client connection lost"):
		// If err is http2 transport ping timeout, retry.
		return true
	case strings.Contains(errorString, "http2: server sent GOAWAY and closed the connection"):
		return true
	case strings.Contains(errorString, "connect: connection refused"):
		// If err is connection refused, retry.
		return true
	case strings.Contains(errorString, "invalid connection"):
		// If err is invalid connection, retry.
		return true
	}

	// If err is EOF, retry.
	var urlErr *url.Error
	if errors.As(err, &urlErr) && strings.Contains(errorString, "EOF") {
		return true
	}

	return false
}

func generateErrorString(err error) string {
	errorString := err.Error()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		errorString = fmt.Sprintf("%s %s", errorString, exitErr.Stderr)
	}
	return errorString
}

func isTransientSqbErr(err error) bool {
	return strings.Contains(err.Error(), "upper: no more rows in")
}

// CheckError is a convenience function to fatally log an exit if the supplied error is non-nil
func CheckError(ctx context.Context, err error) {
	if err != nil {
		logger := logging.RequireLoggerFromContext(ctx)
		logger.WithError(err).WithFatal().Error(ctx, "An error occurred during execution")
	}
}
