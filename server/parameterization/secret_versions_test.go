package parameterization

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clienttesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestLatestSecretResolutionPinsVersionWithoutWritesOrPlaintext(t *testing.T) {
	ctx, s, _, _ := previewFixture(t)
	client := auth.GetKubeClient(ctx)
	immutable := true
	_, err := client.CoreV1().Secrets("lab").Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "one", UID: "one-uid", ResourceVersion: "1"}, Immutable: &immutable, Data: map[string][]byte{"token": []byte("never-return-credential")}}, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().ConfigMaps("lab").Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "index", UID: "source", ResourceVersion: "2"}, Data: map[string]string{"current": `{"name":"one","key":"token"}`}}, metav1.CreateOptions{})
	require.NoError(t, err)
	model := p.Model{SensitivePaths: []string{"/credential"}}
	original := map[string]any{"credential": map[string]any{"latestSecretRef": map[string]any{"configMap": "index", "key": "current"}}}
	values := p.Clone(original)
	versions := map[string]string{}
	fake := client.(interface {
		ClearActions()
		Actions() []clienttesting.Action
		PrependReactor(string, string, clienttesting.ReactionFunc)
	})
	fake.ClearActions()
	require.NoError(t, s.resolveLatestSecrets(ctx, model, values, versions))
	for _, a := range fake.Actions() {
		require.Equal(t, "get", a.GetVerb())
	}
	require.Contains(t, versions, "secret/one")
	require.Contains(t, versions, "latestSecret//credential")
	encoded, _ := json.Marshal([]any{values, versions})
	require.NotContains(t, string(encoded), "never-return-credential")
	require.Contains(t, original["credential"], "latestSecretRef")
	require.Equal(t, map[string]any{"credential": map[string]any{"secretRef": map[string]any{"name": "one", "key": "token"}}}, values)
	// Original retry needs only the pinned Secret, never permission to the index.
	fake.PrependReactor("get", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("not permitted")
	})
	require.NoError(t, validatePinnedSecrets(ctx, "lab", versions))
	require.ErrorContains(t, s.resolveLatestSecrets(ctx, model, p.Clone(original), map[string]string{}), "unavailable or forbidden")
}

func TestLatestSecretResolutionRejectsMutableMissingAndCrossNamespaceVersions(t *testing.T) {
	for _, data := range []string{`{"name":"mutable","key":"token"}`, `{"name":"missing","key":"token"}`, `{"name":"mutable","key":"token","namespace":"other"}`, `{"name":"../escape","key":"token"}`, `private-value`} {
		t.Run(data, func(t *testing.T) {
			ctx, s, _, _ := previewFixture(t)
			client := auth.GetKubeClient(ctx)
			_, err := client.CoreV1().Secrets("lab").Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "mutable"}, Data: map[string][]byte{"token": []byte("private-value")}}, metav1.CreateOptions{})
			require.NoError(t, err)
			_, err = client.CoreV1().ConfigMaps("lab").Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "index"}, Data: map[string]string{"current": data}}, metav1.CreateOptions{})
			require.NoError(t, err)
			values := map[string]any{"credential": map[string]any{"latestSecretRef": map[string]any{"configMap": "index", "key": "current"}}}
			err = s.resolveLatestSecrets(ctx, p.Model{SensitivePaths: []string{"/credential"}}, values, map[string]string{})
			require.Error(t, err)
			require.NotContains(t, err.Error(), "private-value")
		})
	}
}
