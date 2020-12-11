package metrics

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseRequest(t *testing.T) {
	for _, tt := range []struct {
		name     string
		method   string
		url      string
		wantVerb string
		wantKind string
	}{
		{"create", "POST", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases", "Create", "leases"},
		{"list", "GET", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases", "List", "leases"},
		{"watch", "GET", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases?watch=true", "Watch", "leases"},
		{"get", "GET", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases/my-lease", "Get", "leases"},
		{"update", "PUT", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases/my-lease", "Update", "leases"},
		{"delete", "DELETE", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases/my-lease", "Delete", "leases"},
		{"deletecollection", "DELETE", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases", "DeleteCollection", "leases"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			x, _ := url.Parse(tt.url)
			verb, kind := parseRequest(&http.Request{Method: tt.method, URL: x})
			assert.Equal(t, tt.wantVerb, verb)
			assert.Equal(t, tt.wantKind, kind)
		})
	}
}
