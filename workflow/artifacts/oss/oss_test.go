package oss

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestIsTransientOSSErr(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	for _, errCode := range ossTransientErrorCodes {
		err := oss.ServiceError{Code: errCode}
		assert.True(t, isTransientOSSErr(ctx, err))
	}

	err := oss.ServiceError{Code: "NonTransientErrorCode"}
	assert.False(t, isTransientOSSErr(ctx, err))

	nonOSSErr := errors.New("Non-OSS error")
	assert.False(t, isTransientOSSErr(ctx, nonOSSErr))

	assert.False(t, isTransientOSSErr(ctx, nil))
}

// newFakeOSSServer serves GetObject requests with NoSuchKey and ListObjects
// requests with the given object keys under the requested prefix.
func newFakeOSSServer(t *testing.T, keys ...string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("X-Oss-Request-Id", "test")
		if r.URL.Path == "/my-bucket/" || r.URL.Path == "/my-bucket" {
			prefix := r.URL.Query().Get("prefix")
			var contents strings.Builder
			for _, k := range keys {
				if strings.HasPrefix(k, prefix) {
					fmt.Fprintf(&contents, "<Contents><Key>%s</Key><Size>1</Size></Contents>", k)
				}
			}
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><ListBucketResult><Name>my-bucket</Name><Prefix>%s</Prefix><MaxKeys>1</MaxKeys><IsTruncated>false</IsTruncated>%s</ListBucketResult>`, prefix, contents.String())
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message><RequestId>test</RequestId><HostId>127.0.0.1</HostId></Error>`)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOpenStreamNoSuchKey(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tests := []struct {
		name     string
		keys     []string
		wantCode string
	}{
		{name: "missing key", wantCode: argoerrors.CodeNotFound},
		{name: "directory", keys: []string{"my-dir/file.txt"}, wantCode: argoerrors.CodeNotImplemented},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeOSSServer(t, tt.keys...)
			driver := &ArtifactDriver{Endpoint: server.URL, AccessKey: "ak", SecretKey: "sk"}
			art := &wfv1.Artifact{ArtifactLocation: wfv1.ArtifactLocation{OSS: &wfv1.OSSArtifact{
				OSSBucket: wfv1.OSSBucket{Bucket: "my-bucket"},
				Key:       "my-dir",
			}}}
			stream, err := driver.OpenStream(ctx, art)
			assert.Nil(t, stream)
			require.Error(t, err)
			assert.True(t, argoerrors.IsCode(tt.wantCode, err), "unexpected error: %v", err)
		})
	}
}
