package k8s

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseRequest(t *testing.T) {
	for _, tt := range []struct {
		name   string
		method string
		url    string
		want   *kubeRequest
	}{
		// ------------------------------------
		// https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-uris
		// https://kubernetes.io/docs/reference/access-authn-authz/authorization/#determine-the-request-verb
		// ------------------------------------

		// ------------------------------------
		// Cluster-Scoped
		// ------------------------------------
		{
			"cluster-scoped/create",
			"POST",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND/NAME",
			&kubeRequest{Verb: "create", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped/create-subresource",
			"POST",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "create", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"cluster-scoped/delete",
			"DELETE",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND/NAME",
			&kubeRequest{Verb: "delete", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped/deletecollection",
			"DELETE",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND",
			&kubeRequest{Verb: "deletecollection", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"cluster-scoped/get",
			"GET",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND/NAME",
			&kubeRequest{Verb: "get", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped/list",
			"GET",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND",
			&kubeRequest{Verb: "list", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"cluster-scoped/patch",
			"PATCH",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND/NAME",
			&kubeRequest{Verb: "patch", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped/patch-subresource",
			"PATCH",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "patch", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"cluster-scoped/update",
			"PUT",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND/NAME",
			&kubeRequest{Verb: "update", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped/update-subresource",
			"PUT",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "update", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"cluster-scoped/watch-multi",
			"GET",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND?watch=1",
			&kubeRequest{Verb: "watch", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"cluster-scoped/watch-single",
			"GET",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/KIND/NAME?watch=1",
			&kubeRequest{Verb: "watch", Namespace: "", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},

		// ------------------------------------
		// Cluster-Scoped (core / legacy)
		// ------------------------------------
		{
			"cluster-scoped-legacy/create",
			"POST",
			"https://0.0.0.0:65009/api/v1/KIND/NAME",
			&kubeRequest{Verb: "create", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped-legacy/create-subresource",
			"POST",
			"https://0.0.0.0:65009/api/v1/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "create", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"cluster-scoped-legacy/delete",
			"DELETE",
			"https://0.0.0.0:65009/api/v1/KIND/NAME",
			&kubeRequest{Verb: "delete", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped-legacy/deletecollection",
			"DELETE",
			"https://0.0.0.0:65009/api/v1/KIND",
			&kubeRequest{Verb: "deletecollection", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"cluster-scoped-legacy/get",
			"GET",
			"https://0.0.0.0:65009/api/v1/KIND/NAME",
			&kubeRequest{Verb: "get", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped-legacy/list",
			"GET",
			"https://0.0.0.0:65009/api/v1/KIND",
			&kubeRequest{Verb: "list", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"cluster-scoped-legacy/patch",
			"PATCH",
			"https://0.0.0.0:65009/api/v1/KIND/NAME",
			&kubeRequest{Verb: "patch", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped-legacy/patch-subresource",
			"PATCH",
			"https://0.0.0.0:65009/api/v1/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "patch", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"cluster-scoped-legacy/update",
			"PUT",
			"https://0.0.0.0:65009/api/v1/KIND/NAME",
			&kubeRequest{Verb: "update", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"cluster-scoped-legacy/update-subresource",
			"PUT",
			"https://0.0.0.0:65009/api/v1/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "update", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"cluster-scoped-legacy/watch-multi",
			"GET",
			"https://0.0.0.0:65009/api/v1/KIND?watch=1",
			&kubeRequest{Verb: "watch", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"cluster-scoped-legacy/watch-single",
			"GET",
			"https://0.0.0.0:65009/api/v1/KIND/NAME?watch=1",
			&kubeRequest{Verb: "watch", Namespace: "", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},

		// ------------------------------------
		// Namespace-Scoped
		// ------------------------------------
		{
			"namespace-scoped/create",
			"POST",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "create", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped/create-subresource",
			"POST",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "create", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"namespace-scoped/delete",
			"DELETE",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "delete", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped/deletecollection",
			"DELETE",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND",
			&kubeRequest{Verb: "deletecollection", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"namespace-scoped/get",
			"GET",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "get", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped/list",
			"GET",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND",
			&kubeRequest{Verb: "list", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"namespace-scoped/patch",
			"PATCH",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "patch", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped/patch-subresource",
			"PATCH",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "patch", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"namespace-scoped/update",
			"PUT",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "update", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped/update-subresource",
			"PUT",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "update", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"namespace-scoped/watch-multi",
			"GET",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND?watch=1",
			&kubeRequest{Verb: "watch", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"namespace-scoped/watch-single",
			"GET",
			"https://0.0.0.0:65009/apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME?watch=1",
			&kubeRequest{Verb: "watch", Namespace: "NAMESPACE", Group: "GROUP", Version: "VERSION", Kind: "KIND", Name: "NAME", Subresource: ""},
		},

		// ------------------------------------
		// Namespace-Scoped (core / legacy)
		// ------------------------------------
		{
			"namespace-scoped-legacy/create",
			"POST",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "create", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped-legacy/create-subresource",
			"POST",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "create", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"namespace-scoped-legacy/delete",
			"DELETE",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "delete", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped-legacy/deletecollection",
			"DELETE",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND",
			&kubeRequest{Verb: "deletecollection", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"namespace-scoped-legacy/get",
			"GET",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "get", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped-legacy/list",
			"GET",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND",
			&kubeRequest{Verb: "list", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"namespace-scoped-legacy/patch",
			"PATCH",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "patch", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped-legacy/patch-subresource",
			"PATCH",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "patch", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"namespace-scoped-legacy/update",
			"PUT",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND/NAME",
			&kubeRequest{Verb: "update", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
		{
			"namespace-scoped-legacy/update-subresource",
			"PUT",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND/NAME/SUBRESOURCE",
			&kubeRequest{Verb: "update", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: "SUBRESOURCE"},
		},
		{
			"namespace-scoped-legacy/watch-multi",
			"GET",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND?watch=1",
			&kubeRequest{Verb: "watch", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "", Subresource: ""},
		},
		{
			"namespace-scoped-legacy/watch-single",
			"GET",
			"https://0.0.0.0:65009/api/v1/namespaces/NAMESPACE/KIND/NAME?watch=1",
			&kubeRequest{Verb: "watch", Namespace: "NAMESPACE", Group: "", Version: "v1", Kind: "KIND", Name: "NAME", Subresource: ""},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			x, _ := url.Parse(tt.url)
			kubeRequest, err := ParseRequest(&http.Request{Method: tt.method, URL: x})
			assert.NoError(t, err)
			assert.Equal(t, tt.want, kubeRequest)
		})
	}
}
