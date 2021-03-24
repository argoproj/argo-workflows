package k8s

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseRequest(t *testing.T) {
	for _, tt := range []struct {
		name     string
		method   string
		url      string
		wantVerb string
		wantKind string
	}{
		{"create", "POST", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases", "Create", "leases"},
		{"create/short", "POST", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/leases", "Create", "leases"},
		{"create/exec", "POST", "https://0.0.0.0:65009/api/v1/namespaces/argo/pods/my-pod/exec", "Create", "pods/exec"},
		{"list", "GET", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases", "List", "leases"},
		{"watch", "GET", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases?watch=true", "Watch", "leases"},
		{"get", "GET", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases/my-lease", "Get", "leases"},
		{"update", "PUT", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases/my-lease", "Update", "leases"},
		{"update/status", "PUT", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases/my-lease/status", "Update", "leases/status"},
		{"delete", "DELETE", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases/my-lease", "Delete", "leases"},
		{"deletecollection", "DELETE", "https://0.0.0.0:65009/apis/coordination.k8s.io/v1/namespaces/argo/leases", "DeleteCollection", "leases"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			x, _ := url.Parse(tt.url)
			verb, kind := ParseRequest(&http.Request{Method: tt.method, URL: x})
			assert.Equal(t, tt.wantVerb, verb)
			assert.Equal(t, tt.wantKind, kind)
		})
	}
}
