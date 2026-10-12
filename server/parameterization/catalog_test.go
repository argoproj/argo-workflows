package parameterization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestCatalogHTTPCallerScopePaginationAndForbidden(t *testing.T) {
	kinds := map[schema.GroupVersionResource]string{}
	for _, typ := range resourceTypes {
		kinds[schema.GroupVersionResource{Group: p.Group, Version: p.Version, Resource: typ.Resource}] = typ.Kind + "List"
	}
	c := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), kinds,
		object("ParameterTemplate", "visible", "lab", map[string]any{"schema": map[string]any{"type": "object"}}),
		object("ParameterTemplate", "other-namespace", "other", map[string]any{}))
	h := Handler{Gatekeeper: &testGatekeeper{client: c}, Namespace: "lab"}
	get := func(path string, code int) []byte {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, Prefix+path, nil))
		require.Equal(t, code, w.Code, w.Body.String())
		return w.Body.Bytes()
	}
	var types []ResourceType
	require.NoError(t, json.Unmarshal(get("lab/catalog", 200), &types))
	require.Len(t, types, 12)
	var page unstructured.UnstructuredList
	require.NoError(t, json.Unmarshal(get("lab/catalog/parametertemplates", 200), &page))
	require.Len(t, page.Items, 1)
	require.Equal(t, "visible", page.Items[0].GetName())
	get("lab/catalog/parametertemplates?name=visible", 200)
	get("lab/catalog/secrets", 400)
	get("other/catalog/parametertemplates", 403)
	c.PrependReactor("list", "parametertemplates", func(a clienttesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.UnstructuredList{Object: map[string]any{"metadata": map[string]any{"continue": "next"}}}, nil
	})
	require.NoError(t, json.Unmarshal(get("lab/catalog/parametertemplates?continue=opaque-cursor", 200), &page))
	require.Equal(t, "next", page.GetContinue())
	c.PrependReactor("get", "parametertemplates", func(a clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: p.Group, Resource: "parametertemplates"}, "visible", nil)
	})
	get("lab/catalog/parametertemplates?name=visible", 403)
}

func TestCatalogPaginationSentToAPIServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/apis/"+p.Group+"/"+p.Version+"/namespaces/lab/parametertemplates", r.URL.Path)
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "opaque-cursor", r.URL.Query().Get("continue"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apiVersion":"` + p.Group + `/` + p.Version + `","kind":"ParameterTemplateList","metadata":{"continue":"next"},"items":[]}`))
	}))
	defer server.Close()
	client, err := dynamic.NewForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)
	s := Store{Client: client, Namespace: "lab"}
	page, err := s.catalog(context.Background(), "parametertemplates", "", "opaque-cursor")
	require.NoError(t, err)
	require.Equal(t, "next", page.(*unstructured.UnstructuredList).GetContinue())
}

func TestCatalogMetadataUsesCallerTransportAndOmitsDocuments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/apis/"+p.Group+"/"+p.Version+"/namespaces/lab/parametertemplates", r.URL.Path)
		assert.Equal(t, "application/json;as=PartialObjectMetadataList;g=meta.k8s.io;v=v1", r.Header.Get("Accept"))
		assert.Equal(t, "Bearer caller-token", r.Header.Get("Authorization"))
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "next-page", r.URL.Query().Get("continue"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apiVersion":"meta.k8s.io/v1","kind":"PartialObjectMetadataList","metadata":{"continue":"third","resourceVersion":"42"},"items":[{"metadata":{"name":"large","namespace":"lab","uid":"one","resourceVersion":"41","annotations":{"kubectl.kubernetes.io/last-applied-configuration":"do-not-return"}},"spec":{"values":"do-not-return"}}]}`))
	}))
	defer server.Close()
	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL, BearerToken: "caller-token"})
	require.NoError(t, err)
	store := Store{Namespace: "lab", MetadataClient: client.CoreV1().RESTClient()}
	page, err := store.catalogMetadata(context.Background(), "parametertemplates", "next-page")
	require.NoError(t, err)
	require.Equal(t, "third", page.GetContinue())
	require.Equal(t, "42", page.GetResourceVersion())
	require.Equal(t, "large", page.Items[0].GetName())
	require.Equal(t, "one", string(page.Items[0].GetUID()))
	encoded, err := json.Marshal(page)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "do-not-return")
	_, err = store.catalogMetadata(context.Background(), "secrets", "")
	require.Error(t, err)
}

func TestHistorySummaryPreservesCommittedChainAndReadsDetailSeparately(t *testing.T) {
	ctx := context.Background()
	store := testStore()
	saved, err := store.Save(ctx, "draft", SaveRequest{Template: "typed", Values: map[string]any{"count": 2}, Message: "first"})
	require.NoError(t, err)
	first, _, _ := unstructured.NestedString(saved.Object, "spec", "revision")
	_, err = store.Save(ctx, "draft", SaveRequest{Template: "typed", Values: map[string]any{"count": 0}, Message: "invalid draft"})
	require.NoError(t, err)
	list, err := store.History(ctx, "draft", 1, "")
	require.NoError(t, err)
	page := historySummaries(list)
	require.Len(t, page.Items, 1)
	require.Equal(t, list.GetContinue(), page.GetContinue())
	require.Equal(t, int64(1), page.Items[0].Object["spec"].(map[string]any)["validationErrorCount"])
	require.NotContains(t, page.Items[0].Object["spec"], "values")
	require.NotContains(t, page.Items[0].Object["spec"], "model")
	require.NotContains(t, page.Items[0].Object["spec"], "errors")
	older, err := store.History(ctx, "draft", 1, page.GetContinue())
	require.NoError(t, err)
	require.Equal(t, first, older.Items[0].GetName())
	detail, err := store.get(ctx, "parameterrevisions", first)
	require.NoError(t, err)
	require.Contains(t, detail.Object["spec"], "values")
	require.Contains(t, list.Items[0].Object["spec"], "values", "projection must not alter the immutable record")
}
