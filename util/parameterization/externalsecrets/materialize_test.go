package externalsecrets

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type fakeProvider struct {
	meta      Metadata
	fail      bool
	reads     int
	afterRead func()
}

func (f *fakeProvider) Metadata(context.Context) (Metadata, error) {
	if f.fail {
		return Metadata{}, fmt.Errorf("backend unavailable")
	}
	return f.meta, nil
}
func (f *fakeProvider) Read(context.Context, int) ([]byte, error) {
	f.reads++
	if f.afterRead != nil {
		f.afterRead()
	}
	return []byte("synthetic-credential"), nil
}
func TestMaterializationRotationRevocationAndOutage(t *testing.T) {
	provider := &fakeProvider{meta: Metadata{Current: 1, Versions: map[string]Version{"1": {}}}}
	client := fake.NewSimpleClientset()
	now := time.Now().UTC()
	m := Materializer{Client: client, Provider: provider, Namespace: "test", Name: "latest", SourceID: "source", Validity: time.Minute, Now: func() time.Time { return now }}
	require.NoError(t, m.Reconcile(t.Context()))
	require.Equal(t, 1, provider.reads)
	secret, err := client.CoreV1().Secrets("test").Get(t.Context(), "external-source-v1", metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, *secret.Immutable)
	require.NoError(t, p.ValidateExternalSecretLease(secret, now))
	require.Error(t, p.ValidateExternalSecretLease(secret, now.Add(time.Minute)))
	require.NoError(t, m.Reconcile(t.Context()))
	require.Equal(t, 1, provider.reads, "metadata refresh never decrypts old versions")
	provider.meta.Current = 2
	provider.meta.Versions["2"] = Version{}
	require.NoError(t, m.Reconcile(t.Context()))
	index, err := client.CoreV1().ConfigMaps("test").Get(t.Context(), "latest", metav1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, index.Data["current"], "-v2")
	provider.meta.Versions["1"] = Version{Destroyed: true}
	require.NoError(t, m.Reconcile(t.Context()))
	_, err = client.CoreV1().Secrets("test").Get(t.Context(), "external-source-v1", metav1.GetOptions{})
	require.Error(t, err)
	provider.fail = true
	require.Error(t, m.Reconcile(t.Context()))
	_, err = client.CoreV1().Secrets("test").Get(t.Context(), "external-source-v2", metav1.GetOptions{})
	require.NoError(t, err)
	now = now.Add(61 * time.Second)
	require.Error(t, m.Reconcile(t.Context()))
	_, err = client.CoreV1().Secrets("test").Get(t.Context(), "external-source-v2", metav1.GetOptions{})
	require.Error(t, err)
}
func TestMaterializationRaceAndOwnership(t *testing.T) {
	provider := &fakeProvider{meta: Metadata{Current: 1, Versions: map[string]Version{"1": {}}}}
	provider.afterRead = func() { provider.meta.Versions["1"] = Version{Destroyed: true} }
	client := fake.NewSimpleClientset()
	m := Materializer{Client: client, Provider: provider, Namespace: "test", Name: "latest", SourceID: "source", Validity: time.Minute}
	require.ErrorContains(t, m.Reconcile(t.Context()), "changed during")
	secrets, err := client.CoreV1().Secrets("test").List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, secrets.Items)
	provider.afterRead = nil
	provider.meta.Versions["1"] = Version{}
	_, err = client.CoreV1().ConfigMaps("test").Create(t.Context(), &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "latest"}, Data: map[string]string{"untouched": "yes"}}, metav1.CreateOptions{})
	require.NoError(t, err)
	require.ErrorContains(t, m.Reconcile(t.Context()), "owned by another")
}
