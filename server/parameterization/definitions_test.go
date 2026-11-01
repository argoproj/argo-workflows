package parameterization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
)

func TestDefinitionWriteCASStrictValidationAndMetadata(t *testing.T) {
	for resource, kind := range editableDefinitions {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			current := object(kind, "example", "lab", map[string]any{"old": true})
			current.SetUID("one")
			current.SetResourceVersion("7")
			current.SetLabels(map[string]string{"owner": "retain"})
			current.Object["status"] = map[string]any{"observedGeneration": int64(1)}
			original := current.DeepCopy()
			validations, writes := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				assert.Contains(t, r.URL.Path, "/namespaces/lab/"+resource+"/")
				switch r.Method {
				case http.MethodGet:
					if strings.HasSuffix(r.URL.Path, "/absent") {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","reason":"NotFound","code":404}`))
						return
					}
					_ = json.NewEncoder(w).Encode(current)
				case http.MethodPut:
					assert.Equal(t, "Strict", r.URL.Query().Get("fieldValidation"))
					next := &unstructured.Unstructured{}
					if !assert.NoError(t, json.NewDecoder(r.Body).Decode(next)) {
						return
					}
					assert.Equal(t, "7", next.GetResourceVersion())
					assert.Equal(t, "one", string(next.GetUID()))
					if r.URL.Query().Get("dryRun") == "All" {
						validations++
					} else {
						writes++
						next.SetResourceVersion("8")
						current = next.DeepCopy()
					}
					_ = json.NewEncoder(w).Encode(next)
				default:
					t.Errorf("unexpected method: %s", r.Method)
				}
			}))
			defer server.Close()
			client, err := dynamic.NewForConfig(&rest.Config{Host: server.URL})
			require.NoError(t, err)
			s := Store{Client: client, Namespace: "lab"}
			req := DefinitionWrite{Spec: map[string]any{"new": true}, UID: "one", ResourceVersion: "7", ValidateOnly: true}
			if resource == "actiondefinitions" || resource == "workflowactionbindings" {
				_, err = s.writeDefinition(ctx, resource, "example", req)
				require.ErrorContains(t, err, "immutable")
				require.Zero(t, writes)
				require.Zero(t, validations)
				return
			}
			_, err = s.writeDefinition(ctx, resource, "example", req)
			require.NoError(t, err)
			require.Equal(t, 1, validations)
			require.Zero(t, writes)
			require.Equal(t, original.Object, current.Object)
			req.ValidateOnly = false
			got, err := s.writeDefinition(ctx, resource, "example", req)
			require.NoError(t, err)
			require.Equal(t, req.Spec, got.Object["spec"])
			require.Equal(t, original.GetLabels(), got.GetLabels())
			require.JSONEq(t, `{"observedGeneration":1}`, string(mustJSON(got.Object["status"])))
			require.Equal(t, 1, writes)
			for _, pair := range [][2]string{{"", "8"}, {"one", ""}, {"replaced", "8"}, {"one", "7"}} {
				req.UID, req.ResourceVersion = pair[0], pair[1]
				_, err = s.writeDefinition(ctx, resource, "example", req)
				require.True(t, apierrors.IsConflict(err), err)
			}
			_, err = s.writeDefinition(ctx, resource, "absent", DefinitionWrite{Spec: req.Spec, UID: "one", ResourceVersion: "7"})
			require.True(t, apierrors.IsConflict(err))
			require.Equal(t, 1, writes)
		})
	}
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func TestDefinitionCreateDryRunFailureAndWriteRace(t *testing.T) {
	actual, validations := 0, 0
	reject := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","reason":"NotFound","code":404}`))
			return
		}
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Strict", r.URL.Query().Get("fieldValidation"))
		if r.URL.Query().Get("dryRun") == "All" {
			validations++
			if reject {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","reason":"BadRequest","message":"strict schema rejected","code":400}`))
				return
			}
			var obj unstructured.Unstructured
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&obj)) {
				return
			}
			_ = json.NewEncoder(w).Encode(&obj)
			return
		}
		actual++
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","reason":"AlreadyExists","code":409}`))
	}))
	defer server.Close()
	client, err := dynamic.NewForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)
	s := Store{Client: client, Namespace: "lab"}
	ctx := context.Background()
	req := DefinitionWrite{Spec: map[string]any{}, ValidateOnly: true}
	_, err = s.writeDefinition(ctx, "workflowforms", "new", req)
	require.NoError(t, err)
	require.Zero(t, actual)
	req.ValidateOnly = false
	_, err = s.writeDefinition(ctx, "workflowforms", "new", req)
	require.True(t, apierrors.IsAlreadyExists(err))
	require.Equal(t, 1, actual)
	reject = true
	_, err = s.writeDefinition(ctx, "workflowforms", "invalid", req)
	require.True(t, apierrors.IsBadRequest(err), err)
	require.Equal(t, 1, actual)
	require.Equal(t, 3, validations)
}

func TestDefinitionHTTPRejectsReceiptsStatusAndForbiddenCaller(t *testing.T) {
	c := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	h := Handler{Gatekeeper: &testGatekeeper{client: c}, Namespace: "lab"}
	put := func(resource, body string, code int) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodPut, Prefix+"lab/definitions/"+resource+"?name=example", strings.NewReader(body)))
		require.Equal(t, code, w.Code, w.Body.String())
	}
	for _, resource := range []string{"parametersets", "parameterrevisions", "workflowsubmissions", "resourceoperations", "secrets"} {
		put(resource, `{"spec":{}}`, 400)
	}
	put("workflowforms", `{"spec":{},"status":{"phase":"Succeeded"}}`, 400)
	c.PrependReactor("get", "workflowforms", func(a clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "workflowforms"}, "example", nil)
	})
	put("workflowforms", `{"spec":{}}`, 403)
	for _, a := range c.Actions() {
		require.Equal(t, "get", a.GetVerb())
	}
}
