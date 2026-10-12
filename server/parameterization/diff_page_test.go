package parameterization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestDiffPagesPreserveAllLayersAndBindCursor(t *testing.T) {
	changes := []p.Change{}
	for i := range 245 {
		changes = append(changes, p.Change{Op: "replace", Path: fmt.Sprintf("/field%d", i), Before: i, After: i + 1})
	}
	full := map[string]any{"left": "one", "right": "two", "changes": changes, "modelChanges": []p.Change{{Op: "add", Path: "/schema/default", After: false}}, "sourceChanges": []p.Change{{Op: "replace", Path: "/field", Before: "default", After: "user"}}}
	seen := []p.Change{}
	cursor := ""
	for i := range 3 {
		page, err := diffPage(full, DiffRequest{Continue: cursor})
		require.NoError(t, err)
		pagination := page["pagination"].(map[string]any)
		require.Equal(t, 247, pagination["total"])
		require.Equal(t, i*100, pagination["offset"])
		for _, key := range []string{"changes", "modelChanges", "sourceChanges"} {
			seen = append(seen, page[key].([]p.Change)...)
		}
		cursor = pagination["continue"].(string)
		if i < 2 {
			require.NotEmpty(t, cursor)
		}
	}
	require.Empty(t, cursor)
	require.Len(t, seen, 247)
	require.Equal(t, changes, seen[:245])
	require.Len(t, full["changes"], 245, "paging must not mutate the execution comparison")
	first, _ := diffPage(full, DiffRequest{Limit: 1})
	cursor = first["pagination"].(map[string]any)["continue"].(string)
	full["right"] = "another"
	_, err := diffPage(full, DiffRequest{Continue: cursor})
	require.ErrorContains(t, err, "does not match")
	for _, request := range []DiffRequest{{Limit: -1}, {Limit: 201}, {Continue: "garbage"}} {
		_, err = diffPage(full, request)
		require.Error(t, err)
	}
	empty := map[string]any{"changes": []p.Change{}, "modelChanges": []p.Change{}, "sourceChanges": []p.Change{}}
	page, err := diffPage(empty, DiffRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, page["pagination"].(map[string]any)["total"])
}

func TestDiffHTTPRechecksPermissionAndSensitivityOnEveryPage(t *testing.T) {
	s := testStore()
	for i, name := range []string{"first", "second"} {
		_, err := s.resource("parameterrevisions").Create(context.Background(), object("ParameterRevision", name, "lab", Revision{Set: "set", Model: p.Model{Dialect: p.Dialect, Schema: map[string]any{"type": "object"}}, Values: map[string]any{"a": i, "b": i}}), metav1.CreateOptions{})
		require.NoError(t, err)
	}
	h := Handler{Gatekeeper: &testGatekeeper{client: s.Client}, Namespace: "lab"}
	post := func(req DiffRequest, code int) map[string]any {
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodPost, Prefix+"lab/diff", bytes.NewReader(body)))
		require.Equal(t, code, w.Code, w.Body.String())
		var result map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		return result
	}
	first := post(DiffRequest{Left: "first", Right: "second", Limit: 1}, 200)
	require.Len(t, first["changes"], 1)
	cursor := first["pagination"].(map[string]any)["continue"].(string)
	second := post(DiffRequest{Left: "first", Right: "second", Limit: 1, Continue: cursor}, 200)
	require.Len(t, second["changes"], 1)
	s.Client.(*dynamicfake.FakeDynamicClient).PrependReactor("get", "parameterrevisions", func(a clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: p.Group, Resource: "parameterrevisions"}, "first", nil)
	})
	post(DiffRequest{Left: "first", Right: "second", Limit: 1, Continue: cursor}, 403)
}
