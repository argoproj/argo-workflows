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
	i := New(k, "my-mngd-ns", defaultArtifactRepository)
	t.Run("Explicit", func(t *testing.T) {
		ref, err := i.Resolve(&wfv1.ArtifactRepositoryRef{Namespace: "my-ref-ns"}, "my-wf-ns")
		assert.NoError(t, err)
		assert.Equal(t, "my-ref-ns", ref.Namespace)
	})
	t.Run("Explicit.ConfigMapNotFound", func(t *testing.T) {
		ref, err := i.Resolve(&wfv1.ArtifactRepositoryRef{Namespace: "my-ref-ns"}, "my-wf-ns")
		assert.NoError(t, err)
		_, err = i.Get(ref)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
	t.Run("Explicit.ConfigMapMissingKey", func(t *testing.T) {
		_, err := k.CoreV1().ConfigMaps("my-ref-ns").Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "artifact-repositories"},
		})
		assert.NoError(t, err)

		ref, err := i.Resolve(&wfv1.ArtifactRepositoryRef{Namespace: "my-ref-ns"}, "my-wf-ns")
		assert.NoError(t, err)
		_, err = i.Get(ref)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config map missing key")

		err = k.CoreV1().ConfigMaps("my-ref-ns").Delete("artifact-repositories", nil)
		assert.NoError(t, err)
	})
	t.Run("WorkflowNamespaceDefault", func(t *testing.T) {
		_, err := k.CoreV1().ConfigMaps("my-wf-ns").Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "artifact-repositories",
				Annotations: map[string]string{
					"workflows.argoproj.io/default-artifact-repository": "default",
				},
			},
			Data: map[string]string{"default": `s3:
  bucket: my-wf-ns-bucket`},
		})
		assert.NoError(t, err)

		ref, err := i.Resolve(nil, "my-wf-ns")
		if assert.NoError(t, err) {
			assert.Equal(t, "my-wf-ns", ref.Namespace)
		}

		err = k.CoreV1().ConfigMaps("my-wf-ns").Delete("artifact-repositories", nil)
		assert.NoError(t, err)
	})
	t.Run("ManagedNamespaceDefault", func(t *testing.T) {
		_, err := k.CoreV1().ConfigMaps("my-mngd-ns").Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "artifact-repositories",
				Annotations: map[string]string{
					"workflows.argoproj.io/default-artifact-repository": "default",
				},
			},
			Data: map[string]string{"default": `s3:
  bucket: my-mngd-ns-bucket`},
		})
		assert.NoError(t, err)

		ref, err := i.Resolve(nil, "my-wf-ns")
		if assert.NoError(t, err) {
			assert.Equal(t, "my-mngd-ns", ref.Namespace)
		}

		err = k.CoreV1().ConfigMaps("my-mngd-ns").Delete("artifact-repositories", nil)
		assert.NoError(t, err)
	})
	t.Run("Default", func(t *testing.T) {
		ref, err := i.Resolve(nil, "my-wf-ns")
		assert.NoError(t, err)
		assert.Equal(t, wfv1.DefaultArtifactRepositoryRef, ref)
	})
}
