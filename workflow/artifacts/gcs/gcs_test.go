package gcs

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"testing"

	"google.golang.org/api/googleapi"

	argoErrors "github.com/argoproj/argo-workflows/v3/errors"
)

type tlsHandshakeTimeoutError struct{}

func (tlsHandshakeTimeoutError) Temporary() bool { return true }
func (tlsHandshakeTimeoutError) Error() string   { return "net/http: TLS handshake timeout" }

func TestIsTransientGCSErr(t *testing.T) {
	for _, test := range []struct {
		err         error
		shouldretry bool
	}{
		{&googleapi.Error{Code: 0}, false},
		{argoErrors.New(argoErrors.CodeNotFound, "no results for key: foo/bar"), false},
		{&googleapi.Error{Code: 429}, true},
		{&googleapi.Error{Code: 504}, true},
		{&googleapi.Error{Code: 518}, true},
		{&googleapi.Error{Code: 599}, true},
		{&url.Error{Op: "blah", URL: "blah", Err: errors.New("connection refused")}, true},
		{&url.Error{Op: "blah", URL: "blah", Err: errors.New("compute: Received 504 `Gateway Timeout\n`")}, true},
		{&url.Error{Op: "blah", URL: "blah", Err: errors.New("http2: client connection lost")}, true},
		{io.ErrUnexpectedEOF, true},
		{&tlsHandshakeTimeoutError{}, true},
		{fmt.Errorf("Test unwrapping of a temporary error: %w", &googleapi.Error{Code: 500}), true},
		{fmt.Errorf("Test unwrapping of a non-retriable error: %w", &googleapi.Error{Code: 400}), false},
		{fmt.Errorf("writer close: Post \"https://storage.googleapis.com/upload/storage/v1/b/bucket/o?alt=json&name=test.json&uploadType=multipart\": compute: Received 504 `Gateway Timeout\n`"), true},
		{fmt.Errorf("http2: client connection lost"), true},
	} {
		got := isTransientGCSErr(test.err)
		if got != test.shouldretry {
			t.Errorf("%+v: got %v, want %v", test, got, test.shouldretry)
		}
	}
}
