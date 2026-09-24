package namespacedefaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func configMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: ConfigMapName},
		Data:       data,
	}
}

func TestNamespaceDefaults(t *testing.T) {
	t.Run("NoConfigMap", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())

		wf, err := New(kubefake.NewClientset()).Get(ctx, "my-ns")
		require.NoError(t, err)
		assert.Nil(t, wf, "a namespace without the ConfigMap configures no defaults")
	})

	t.Run("Defaults", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		k := kubefake.NewClientset()
		_, err := k.CoreV1().ConfigMaps("my-ns").Create(ctx, configMap(map[string]string{
			Key: "spec:\n  serviceAccountName: my-sa\n  entrypoint: my-entrypoint\n",
		}), metav1.CreateOptions{})
		require.NoError(t, err)

		wf, err := New(k).Get(ctx, "my-ns")
		require.NoError(t, err)
		require.NotNil(t, wf)
		assert.Equal(t, "my-sa", wf.Spec.ServiceAccountName)
		assert.Equal(t, "my-entrypoint", wf.Spec.Entrypoint)
	})
}

func TestNamespaceDefaultsFailLoudly(t *testing.T) {
	t.Run("MissingKey", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		k := kubefake.NewClientset()
		_, err := k.CoreV1().ConfigMaps("my-ns").Create(ctx, configMap(map[string]string{
			"workflowDefault": "spec:\n  entrypoint: my-entrypoint\n",
		}), metav1.CreateOptions{})
		require.NoError(t, err)

		_, err = New(k).Get(ctx, "my-ns")
		require.Error(t, err, "a ConfigMap with a mistyped key must fail rather than silently apply nothing")
		assert.Contains(t, err.Error(), "missing key")
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		k := kubefake.NewClientset()
		_, err := k.CoreV1().ConfigMaps("my-ns").Create(ctx, configMap(map[string]string{
			Key: "spec: [this is not a workflow spec",
		}), metav1.CreateOptions{})
		require.NoError(t, err)

		_, err = New(k).Get(ctx, "my-ns")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})
}
