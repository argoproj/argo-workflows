package parameterization

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestPinnedSecretRetryChecksUIDAndIgnoresMetadataOnlyVersion(t *testing.T) {
	immutable := true
	client := kubefake.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "pinned", Namespace: "lab", UID: "same-secret", ResourceVersion: "new-metadata"}, Immutable: &immutable})
	ctx := context.WithValue(context.Background(), auth.KubeKey, client)
	require.NoError(t, validatePinnedSecrets(ctx, "lab", map[string]string{"secret/pinned": "same-secret:old-metadata"}))
	require.ErrorContains(t, validatePinnedSecrets(ctx, "lab", map[string]string{"secret/pinned": "other-uid:old-metadata"}), "identity changed")
	require.ErrorContains(t, validatePinnedSecrets(ctx, "other", map[string]string{"secret/pinned": "same-secret:old-metadata"}), "unavailable")
	require.NoError(t, client.CoreV1().Secrets("lab").Delete(ctx, "pinned", metav1.DeleteOptions{}))
	require.ErrorContains(t, validatePinnedSecrets(ctx, "lab", map[string]string{"secret/pinned": "same-secret:old-metadata"}), "revoked")
}

func TestSecretWritesAreVersionedIdempotentAndNeverSerialized(t *testing.T) {
	client := kubefake.NewSimpleClientset()
	ctx := context.WithValue(context.Background(), auth.KubeKey, client)
	backend := EncryptedKubernetesSecrets{}
	purpose := SecretWriteContext{"lab", "alice", "typed", "/credential", "a-random-unique-request"}
	value := []byte("synthetic-secret-never-return-this")
	ref, err := backend.Write(ctx, purpose, value)
	require.NoError(t, err)
	encoded, err := json.Marshal(ref)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), string(value))
	repeat, err := backend.Write(ctx, purpose, value)
	require.NoError(t, err)
	require.Equal(t, ref, repeat)
	_, err = backend.Write(ctx, purpose, []byte("replacement"))
	require.True(t, apierrors.IsConflict(err))
	purpose.RequestID = "replacement-new-version"
	replacement, err := backend.Write(ctx, purpose, []byte("replacement"))
	require.NoError(t, err)
	require.NotEqual(t, ref, replacement)
	purpose.Namespace = "other"
	other, err := backend.Write(ctx, purpose, []byte("replacement"))
	require.NoError(t, err)
	require.NotEqual(t, replacement, other)
	secrets, err := client.CoreV1().Secrets("lab").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, secrets.Items, 2)
	for _, secret := range secrets.Items {
		require.True(t, *secret.Immutable)
		metadata, _ := json.Marshal(secret.ObjectMeta)
		require.NotContains(t, string(metadata), string(value))
	}
}

func TestSecretBackendSanitizesProviderFailuresAndUsesCallerAuthorization(t *testing.T) {
	client := kubefake.NewSimpleClientset()
	ctx := context.WithValue(context.Background(), auth.KubeKey, client)
	client.PrependReactor("create", "secrets", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("request body includes sensitive-plaintext")
	})
	backend := EncryptedKubernetesSecrets{}
	purpose := SecretWriteContext{"lab", "alice", "typed", "/credential", "a-random-unique-request"}
	_, err := backend.Write(ctx, purpose, []byte("sensitive-plaintext"))
	require.ErrorContains(t, err, "backend unavailable")
	require.NotContains(t, err.Error(), "sensitive-plaintext")
	client.PrependReactor("get", "secrets", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "name", errors.New("sensitive-plaintext"))
	})
	_, err = backend.Write(ctx, purpose, []byte("sensitive-plaintext"))
	require.True(t, apierrors.IsForbidden(err))
	require.NotContains(t, err.Error(), "sensitive-plaintext")
}

func TestSecretDedicatedEndpointValidatesLogicalValueAndTransport(t *testing.T) {
	ctx, s, _, _ := previewFixture(t)
	model, err := s.get(ctx, "parametertemplates", "typed")
	require.NoError(t, err)
	var m p.Model
	require.NoError(t, decode(model, &m))
	m.SensitivePaths = []string{"/credential"}
	m.SecretSchemas = map[string]map[string]any{"/credential": {"type": "string", "minLength": 8}}
	_, err = s.resource("parametertemplates").Update(ctx, object("ParameterTemplate", "typed", "lab", m), metav1.UpdateOptions{})
	require.NoError(t, err)
	h := Handler{SecretWriter: EncryptedKubernetesSecrets{}}
	req := SecretWriteRequest{Template: "typed", Path: "/credential", RequestID: "a-random-unique-request", Value: "tiny"}
	_, err = h.writeSecret(ctx, s, req)
	require.ErrorContains(t, err, "declared schema")
	req.Value = "synthetic-secret"
	_, err = h.writeSecret(ctx, s, req)
	require.NoError(t, err)
	req.Path = "/not-declared"
	_, err = h.writeSecret(ctx, s, req)
	require.ErrorContains(t, err, "declared sensitive path")
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "http://localhost/secrets", nil)
	r.RemoteAddr = "127.0.0.1:4567"
	require.False(t, h.secretTransportAllowed(r))
	h.AllowLoopbackSecretInput = true
	require.True(t, h.secretTransportAllowed(r))
	r.RemoteAddr = "192.0.2.1:4567"
	r.Header.Set("X-Forwarded-Proto", "https")
	require.False(t, h.secretTransportAllowed(r))
}
