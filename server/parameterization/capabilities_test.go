package parameterization

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestCapabilityDiscoveryDoesNotOverpromiseExecution(t *testing.T) {
	h := Handler{Gatekeeper: &testGatekeeper{client: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())}, Namespace: "lab"}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, Prefix+"lab/capabilities", nil))
	require.Equal(t, 200, w.Code)
	var c CapabilityReport
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &c))
	require.True(t, c.Experimental)
	require.Equal(t, "Argo", c.NativeWorkflowOwner)
	require.Contains(t, c.ActionCallScopes, "namedStepsCalls")
	require.Contains(t, c.PartialRetryScopes, "stableIdentityCompositeLoops")
	require.Contains(t, c.Unsupported, "partialRetryTemplateTopologyChanges")
	require.Contains(t, c.Features, "actionArtifactMappings")
	require.Contains(t, c.PartialRetryScopes, "replaceApprovedLoopExpansion")
	require.Contains(t, c.Unsupported, "encryptedInput")
	require.False(t, c.EncryptedSecretInput)
	enabled := capabilities(true)
	require.NotContains(t, enabled.Unsupported, "encryptedInput")
	require.Contains(t, enabled.Features, "externalVaultVersionMaterialization")
	require.Contains(t, enabled.Unsupported, "perAttemptSecretLease")
	require.Contains(t, enabled.Features, "authorReviewedActionReplacement")
	require.Contains(t, enabled.Unsupported, "automaticActionEffectProof")
	require.True(t, enabled.EncryptedSecretInput)
}
