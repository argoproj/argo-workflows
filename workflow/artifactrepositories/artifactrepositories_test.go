package artifactrepositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo/config"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestArtifactRepositories(t *testing.T) {
	defaultArtifactRepository := &config.ArtifactRepository{}
	k := kubefake.NewSimpleClientset()
	i := New(k, "my-ctrl-ns", defaultArtifactRepository)
	t.Run("Explicit.WorkflowNamespace", func(t *testing.T) {
		_, err := k.CoreV1().ConfigMaps("my-wf-ns").Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "artifact-repositories"},
			Data:       map[string]string{"my-key": ""},
		})
		assert.NoError(t, err)

		ref, err := i.Resolve(&wfv1.ArtifactRepositoryRef{Key: "my-key"}, "my-wf-ns")
		if assert.NoError(t, err) {
			assert.Equal(t, "my-wf-ns", ref.Namespace)
			assert.Equal(t, "artifact-repositories", ref.ConfigMap)
			assert.Equal(t, "my-key", ref.Key)
			assert.False(t, ref.Default)
		}

		err = k.CoreV1().ConfigMaps("my-wf-ns").Delete("artifact-repositories", nil)
		assert.NoError(t, err)
	})
	t.Run("Explicit.ControllerNamespace", func(t *testing.T) {
		_, err := k.CoreV1().ConfigMaps("my-ctrl-ns").Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "artifact-repositories"},
			Data:       map[string]string{"my-key": ""},
		})
		assert.NoError(t, err)

		ref, err := i.Resolve(&wfv1.ArtifactRepositoryRef{Key: "my-key"}, "my-wf-ns")
		if assert.NoError(t, err) {
			assert.Equal(t, "my-ctrl-ns", ref.Namespace)
			assert.Equal(t, "artifact-repositories", ref.ConfigMap)
			assert.Equal(t, "my-key", ref.Key)
			assert.False(t, ref.Default)
		}

		err = k.CoreV1().ConfigMaps("my-ctrl-ns").Delete("artifact-repositories", nil)
		assert.NoError(t, err)
	})
	t.Run("Explicit.ConfigMapNotFound", func(t *testing.T) {
		_, err := i.Resolve(&wfv1.ArtifactRepositoryRef{}, "my-wf-ns")
		assert.Error(t, err)
	})
	t.Run("Explicit.ConfigMapMissingKey", func(t *testing.T) {
		_, err := k.CoreV1().ConfigMaps("my-ns").Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "artifact-repositories"},
		})
		assert.NoError(t, err)

		_, err = i.Resolve(&wfv1.ArtifactRepositoryRef{}, "my-wf-ns")
		assert.Error(t, err)

		err = k.CoreV1().ConfigMaps("my-ns").Delete("artifact-repositories", nil)
		assert.NoError(t, err)
	})
	t.Run("WorkflowNamespaceDefault", func(t *testing.T) {
		_, err := k.CoreV1().ConfigMaps("my-wf-ns").Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "artifact-repositories",
				Annotations: map[string]string{"workflows.argoproj.io/default-artifact-repository": "default-v1"},
			},
			Data: map[string]string{"default-v1": ""},
		})
		assert.NoError(t, err)

		ref, err := i.Resolve(nil, "my-wf-ns")
		if assert.NoError(t, err) {
			assert.Equal(t, "my-wf-ns", ref.Namespace)
			assert.Equal(t, "artifact-repositories", ref.ConfigMap)
			assert.Equal(t, "default-v1", ref.Key)
			assert.False(t, ref.Default)
		}
		err = k.CoreV1().ConfigMaps("my-wf-ns").Delete("artifact-repositories", nil)
		assert.NoError(t, err)
	})
	t.Run("Default", func(t *testing.T) {
		ref, err := i.Resolve(nil, "my-wf-ns")
		assert.NoError(t, err)
		assert.Equal(t, wfv1.DefaultArtifactRepositoryRefStatus, ref)
	})
}
