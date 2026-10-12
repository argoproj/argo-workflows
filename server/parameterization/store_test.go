package parameterization

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func testStore() Store {
	model := p.Model{Dialect: p.Dialect, Schema: map[string]any{"type": "object", "properties": map[string]any{"enabled": map[string]any{"type": "boolean", "default": true}, "count": map[string]any{"type": "integer", "minimum": 1}}, "required": []any{"count"}}}
	c := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), object("ParameterTemplate", "typed", "lab", model))
	return Store{Client: c, Namespace: "lab", Actor: "test-user"}
}
func TestSaveHistoryDraftValidationAndRestore(t *testing.T) {
	s := testStore()
	ctx := context.Background()
	first, err := s.Save(ctx, "my-set", SaveRequest{Template: "typed", Values: map[string]any{"count": 2, "enabled": false}, Message: "initial"})
	require.NoError(t, err)
	head, _, _ := unstructured.NestedString(first.Object, "spec", "revision")
	second, err := s.Save(ctx, "my-set", SaveRequest{Template: "typed", Values: map[string]any{"count": 2, "enabled": false}})
	require.NoError(t, err)
	require.Equal(t, first.Object, second.Object, "same content is idempotent")
	_, err = s.Save(ctx, "my-set", SaveRequest{Template: "typed", Values: map[string]any{"count": 0, "enabled": false}})
	require.NoError(t, err, "invalid drafts are allowed, not runnable")
	history, err := s.History(ctx, "my-set", 1, "")
	require.NoError(t, err)
	require.Len(t, history.Items, 1)
	require.NotEmpty(t, history.GetContinue())
	var rev Revision
	require.NoError(t, decode(&history.Items[0], &rev))
	require.NotEmpty(t, rev.Errors)
	older, err := s.History(ctx, "my-set", 1, history.GetContinue())
	require.NoError(t, err)
	require.Equal(t, head, older.Items[0].GetName())
	_, err = s.Save(ctx, "other", SaveRequest{Template: "typed", Values: map[string]any{"count": 4}, RestoredFrom: head})
	require.ErrorContains(t, err, "another parameter set")
	_, err = s.Save(ctx, "my-set", SaveRequest{Template: "typed", Values: map[string]any{"count": 2, "enabled": false}, RestoredFrom: head})
	require.NoError(t, err)
	history, err = s.History(ctx, "my-set", 25, "")
	require.NoError(t, err)
	require.Len(t, history.Items, 3)
	require.NoError(t, decode(&history.Items[0], &rev))
	require.Equal(t, head, rev.RestoredFrom)
	require.Equal(t, "test-user", rev.Actor)
}
func TestOptimisticConcurrencyDoesNotLoseCommittedDraft(t *testing.T) {
	s := testStore()
	ctx := context.Background()
	_, err := s.Save(ctx, "my-set", SaveRequest{Template: "typed", Values: map[string]any{"count": 1}})
	require.NoError(t, err)
	current, err := s.get(ctx, "parametersets", "my-set")
	require.NoError(t, err)
	current.SetResourceVersion("7")
	_, err = s.resource("parametersets").Update(ctx, current, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.Save(ctx, "my-set", SaveRequest{Template: "typed", Values: map[string]any{"count": 2}, ResourceVersion: "6"})
	require.True(t, apierrors.IsConflict(err))
	// A competing write after reading the head must still be caught by Kubernetes CAS.
	s.Client.(*dynamicfake.FakeDynamicClient).PrependReactor("update", "parametersets", func(a clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewConflict(schema.GroupResource{Group: p.Group, Resource: "parametersets"}, "my-set", nil)
	})
	_, err = s.Save(ctx, "my-set", SaveRequest{Template: "typed", Values: map[string]any{"count": 2}, ResourceVersion: "7"})
	require.True(t, apierrors.IsConflict(err))
	history, err := s.History(ctx, "my-set", 25, "")
	require.NoError(t, err)
	require.Len(t, history.Items, 1, "orphan revision is not committed history")
}
func TestReadRejectsLargeUnknownAndMultipleRequests(t *testing.T) {
	for _, body := range []string{`{"template":"typed","unknown":"value"}`, `{} {}`, `{"message":"` + strings.Repeat("a", 1024*1024) + `"}`} {
		var req SaveRequest
		require.Error(t, read(httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/", strings.NewReader(body)), &req))
	}
}
