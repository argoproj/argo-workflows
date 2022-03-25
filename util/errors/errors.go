package errors

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"

	argoerrs "github.com/argoproj/argo-workflows/v3/errors"
)

func IsTransientErr(err error) bool {
	if err == nil {
		return false
	}
	err = argoerrs.Cause(err)
	isTransient := isExceededQuotaErr(err) || apierr.IsTooManyRequests(err) || isResourceQuotaConflictErr(err) || isTransientNetworkErr(err) || apierr.IsServerTimeout(err) || apierr.IsServiceUnavailable(err) || matchTransientErrPattern(err) ||
		errors.Is(err, NewErrTransient(""))
	if isTransient {
		log.Infof("Transient error: %v", err)
	} else {
		log.Warnf("Non-transient error: %v", err)
	}
	return isTransient
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
		}
	}

	errorString := generateErrorString(err)
	if strings.Contains(errorString, "net/http: TLS handshake timeout") {
		// If error is - tlsHandshakeTimeoutError, retry.
		return true
	} else if strings.Contains(errorString, "i/o timeout") {
		// If error is - tcp timeoutError, retry.
		return true
	} else if strings.Contains(errorString, "connection timed out") {
		// If err is a net.Dial timeout, retry.
		return true
	}

	return false
}

func generateErrorString(err error) string {
	errorString := err.Error()
	if exitErr, ok := err.(*exec.ExitError); ok {
		errorString = fmt.Sprintf("%s %s", errorString, exitErr.Stderr)
	}
	return errorString
}
