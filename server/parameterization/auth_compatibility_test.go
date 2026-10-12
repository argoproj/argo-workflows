package parameterization

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4/server/auth"
	servertypes "github.com/argoproj/argo-workflows/v4/server/types"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Exercise the public HTTP handler, real authentication and Kubernetes clients.
// Only the Kubernetes HTTP service is replaced to model an older API server.
func TestCatalogWithLegacyTokenReview(t *testing.T) {
	for _, tc := range []struct {
		name, authorization, username                                string
		selfStatus, reviewStatus, wantStatus, wantReviews, wantReads int
		authenticated, denyResource                                  bool
		discovery                                                    int
	}{
		{"valid-cached", "Bearer caller", "caller", 404, 200, 200, 1, 2, true, false, 0},
		{"resource-denied", "Bearer caller", "caller", 404, 200, 403, 1, 2, true, true, 0},
		{"invalid-not-cached", "Bearer caller", "", 404, 200, 403, 2, 0, false, false, 0},
		{"anonymous-rejected", "Bearer caller", "system:anonymous", 404, 200, 403, 2, 0, true, false, 0},
		{"review-denied", "Bearer caller", "caller", 404, 403, 403, 2, 0, true, false, 0},
		{"self-review-denied", "Bearer caller", "caller", 403, 200, 403, 0, 0, true, false, 0},
		{"forbidden-missing-api", "Bearer caller", "caller", 403, 200, 200, 1, 2, true, false, 1},
		{"missing-api-invalid-token", "Bearer caller", "", 403, 200, 403, 2, 0, false, false, 1},
		{"discovery-failed", "Bearer caller", "caller", 403, 200, 403, 0, 0, true, false, 2},
		{"self-review-unauthorized", "Bearer caller", "caller", 401, 200, 403, 0, 0, true, false, 0},
		{"self-review-unavailable", "Bearer caller", "caller", 503, 200, 403, 0, 0, true, false, 0},
		{"empty-token", "Bearer ", "caller", 404, 200, 403, 0, 0, true, false, 0},
		{"basic-not-converted", "Basic dXNlcjpwYXNz", "caller", 404, 200, 403, 0, 0, true, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reviews, reads := 0, 0
			status := func(w http.ResponseWriter, code int) {
				w.WriteHeader(code)
				reason := map[int]metav1.StatusReason{401: metav1.StatusReasonUnauthorized, 403: metav1.StatusReasonForbidden, 404: metav1.StatusReasonNotFound, 503: metav1.StatusReasonServiceUnavailable}[code]
				_ = json.NewEncoder(w).Encode(metav1.Status{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: "Failure", Code: int32(code), Reason: reason})
			}
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/apis/authentication.k8s.io/v1/selfsubjectreviews":
					status(w, tc.selfStatus)
				case "/apis/authentication.k8s.io/v1":
					if tc.discovery == 2 {
						status(w, 503)
						return
					}
					resources := []metav1.APIResource{{Name: "tokenreviews"}}
					if tc.discovery != 1 {
						resources = append(resources, metav1.APIResource{Name: "selfsubjectreviews"})
					}
					_ = json.NewEncoder(w).Encode(metav1.APIResourceList{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "APIResourceList"}, GroupVersion: "authentication.k8s.io/v1", APIResources: resources})
				case "/apis/authentication.k8s.io/v1/tokenreviews":
					reviews++
					assert.Equal(t, "Bearer reviewer", r.Header.Get("Authorization"))
					var review authenticationv1.TokenReview
					raw, readErr := io.ReadAll(r.Body)
					assert.NoError(t, readErr)
					_, _, decodeErr := scheme.Codecs.UniversalDeserializer().Decode(raw, nil, &review)
					assert.NoError(t, decodeErr)
					assert.Equal(t, "caller", review.Spec.Token)
					if tc.reviewStatus != 200 {
						status(w, tc.reviewStatus)
						return
					}
					_ = json.NewEncoder(w).Encode(authenticationv1.TokenReview{TypeMeta: metav1.TypeMeta{APIVersion: "authentication.k8s.io/v1", Kind: "TokenReview"}, Status: authenticationv1.TokenReviewStatus{Authenticated: tc.authenticated, User: authenticationv1.UserInfo{Username: tc.username}}})
				case "/apis/parameters.workflows.example.io/v1alpha1/namespaces/lab/parametertemplates":
					reads++
					assert.Equal(t, "Bearer caller", r.Header.Get("Authorization"), "resource requests must retain caller identity")
					if tc.denyResource {
						status(w, 403)
						return
					}
					_, _ = w.Write([]byte(`{"apiVersion":"parameters.workflows.example.io/v1alpha1","kind":"ParameterTemplateList","items":[]}`))
				default:
					status(w, 404)
				}
			}))
			defer backend.Close()
			configFile := filepath.Join(t.TempDir(), "config")
			require.NoError(t, os.WriteFile(configFile, []byte(fmt.Sprintf("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: %s\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: test\ncurrent-context: test\nusers:\n- name: test\n  user: {}\n", backend.URL)), 0600))
			t.Setenv("KUBECONFIG", configFile)
			config := &rest.Config{Host: backend.URL, BearerToken: "reviewer"}
			gate, err := auth.NewGatekeeper(auth.Modes{auth.Client: true}, &servertypes.Clients{Kubernetes: kubernetes.NewForConfigOrDie(config)}, config, nil, auth.DefaultClientForAuthorization, "", "", false, nil, time.Minute)
			require.NoError(t, err)
			handler := Handler{Gatekeeper: gate, Namespace: "lab"}
			for range 2 {
				request := httptest.NewRequestWithContext(logging.TestContext(t.Context()), http.MethodGet, Prefix+"lab/catalog/parametertemplates", nil)
				request.Header.Set("Authorization", tc.authorization)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				require.Equal(t, tc.wantStatus, response.Code, response.Body.String())
			}
			require.Equal(t, tc.wantReviews, reviews)
			require.Equal(t, tc.wantReads, reads)
		})
	}
}
