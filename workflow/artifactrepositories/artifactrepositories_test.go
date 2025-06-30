package artifactrepositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestArtifactRepositories(t *testing.T) {
	defaultArtifactRepository := &wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{KeyFormat: "foo"},
	}
	defaultArtifactRepositoryRefStatus := &wfv1.ArtifactRepositoryRefStatus{
		Default:            true,
		ArtifactRepository: defaultArtifactRepository,
	}
	k := kubefake.NewSimpleClientset()
	i := New(k, "my-ctrl-ns", defaultArtifactRepository)
	t.Run("Explicit.WorkflowNamespace", func(t *testing.T) {
		ctx := context.Background()
		ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		_, err := k.CoreV1().ConfigMaps("my-wf-ns").Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "artifact-repositories"},
			Data: map[string]string{"my-key": `
s3:
  keyFormat: bar
`},
		}, metav1.CreateOptions{})
		require.NoError(t, err)

		ref, err := i.Resolve(ctx, &wfv1.ArtifactRepositoryRef{Key: "my-key"}, "my-wf-ns")
		require.NoError(t, err)
		assert.Equal(t, "my-wf-ns", ref.Namespace)
		assert.Equal(t, "artifact-repositories", ref.ConfigMap)
		assert.Equal(t, "my-key", ref.Key)
		assert.False(t, ref.Default)
		assert.NotNil(t, ref.ArtifactRepository)

		repo, err := i.Get(ctx, ref)
		require.NoError(t, err)
		assert.Equal(t, &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{KeyFormat: "bar"}}, repo)

		err = k.CoreV1().ConfigMaps("my-wf-ns").Delete(ctx, "artifact-repositories", metav1.DeleteOptions{})
		require.NoError(t, err)
	})
	t.Run("Explicit.ControllerNamespace", func(t *testing.T) {
		ctx := context.Background()
		ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		_, err := k.CoreV1().ConfigMaps("my-ctrl-ns").Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "artifact-repositories"},
			Data: map[string]string{"my-key": `
s3:
  keyFormat: bar
`},
		}, metav1.CreateOptions{})
		require.NoError(t, err)

		ref, err := i.Resolve(ctx, &wfv1.ArtifactRepositoryRef{Key: "my-key"}, "my-wf-ns")
		require.NoError(t, err)
		assert.Equal(t, "my-ctrl-ns", ref.Namespace)
		assert.Equal(t, "artifact-repositories", ref.ConfigMap)
		assert.Equal(t, "my-key", ref.Key)
		assert.False(t, ref.Default)
		assert.NotNil(t, ref.ArtifactRepository)

		repo, err := i.Get(ctx, ref)
		require.NoError(t, err)
		assert.Equal(t, &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{KeyFormat: "bar"}}, repo)

		err = k.CoreV1().ConfigMaps("my-ctrl-ns").Delete(ctx, "artifact-repositories", metav1.DeleteOptions{})
		require.NoError(t, err)
	})
	t.Run("Explicit.ConfigMapNotFound", func(t *testing.T) {
		ctx := context.Background()
		ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		_, err := i.Resolve(ctx, &wfv1.ArtifactRepositoryRef{}, "my-wf-ns")
		require.Error(t, err)
	})
	t.Run("Explicit.ConfigMapMissingKey", func(t *testing.T) {
		ctx := context.Background()
		ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		_, err := k.CoreV1().ConfigMaps("my-ns").Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "artifact-repositories"},
		}, metav1.CreateOptions{})
		require.NoError(t, err)

		_, err = i.Resolve(ctx, &wfv1.ArtifactRepositoryRef{}, "my-wf-ns")
		require.Error(t, err)

		err = k.CoreV1().ConfigMaps("my-ns").Delete(ctx, "artifact-repositories", metav1.DeleteOptions{})
		require.NoError(t, err)
	})
	t.Run("WorkflowNamespaceDefault", func(t *testing.T) {
		ctx := context.Background()
		ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		_, err := k.CoreV1().ConfigMaps("my-wf-ns").Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "artifact-repositories",
				Annotations: map[string]string{"workflows.argoproj.io/default-artifact-repository": "default-v1"},
			},
			Data: map[string]string{"default-v1": `
s3:
  keyFormat: bar
`},
		}, metav1.CreateOptions{})
		require.NoError(t, err)

		ref, err := i.Resolve(ctx, nil, "my-wf-ns")
		require.NoError(t, err)
		assert.Equal(t, "my-wf-ns", ref.Namespace)
		assert.Equal(t, "artifact-repositories", ref.ConfigMap)
		assert.Equal(t, "default-v1", ref.Key)
		assert.False(t, ref.Default)
		assert.NotNil(t, ref.ArtifactRepository)

		repo, err := i.Get(ctx, ref)
		require.NoError(t, err)
		assert.Equal(t, &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{KeyFormat: "bar"}}, repo)

		err = k.CoreV1().ConfigMaps("my-wf-ns").Delete(ctx, "artifact-repositories", metav1.DeleteOptions{})
		require.NoError(t, err)
	})
	t.Run("DefaultWithNamespace", func(t *testing.T) {
		ctx := context.Background()
		ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		_, err := k.CoreV1().ConfigMaps("my-wf-ns").Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "artifact-repositories",
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)

		ref, err := i.Resolve(ctx, nil, "my-wf-ns")
		require.NoError(t, err)
		assert.Equal(t, defaultArtifactRepositoryRefStatus, ref)

		err = k.CoreV1().ConfigMaps("my-wf-ns").Delete(ctx, "artifact-repositories", metav1.DeleteOptions{})
		require.NoError(t, err)
	})
	t.Run("Default", func(t *testing.T) {
		ctx := context.Background()
		ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		ref, err := i.Resolve(ctx, nil, "my-wf-ns")
		require.NoError(t, err)
		assert.Equal(t, defaultArtifactRepositoryRefStatus, ref)
	})
}
