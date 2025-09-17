package sync

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/server/auth"
)

func withKubeClient(kubeClient *fake.Clientset) context.Context {
	return context.WithValue(context.Background(), auth.KubeKey, kubeClient)
}

func Test_syncServer_CreateSyncLimit(t *testing.T) {
	t.Run("SizeLimit <= 0", func(t *testing.T) {
		ctx := context.Background()
		server := NewSyncServer(ctx, &fake.Clientset{}, "", nil)

		req := &syncpkg.CreateSyncLimitRequest{
			Name:      "test-cm",
			Namespace: "test-ns",
			Key:       "test-key",
			SizeLimit: 0,
		}

		_, err := server.CreateSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, statusErr.Code())
		require.Contains(t, statusErr.Message(), "size limit must be greater than zero")
	})

	t.Run("Error creating ConfigMap", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()

		kubeClient.PrependReactor("create", "configmaps", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(
				schema.GroupResource{Group: "", Resource: "configmaps"},
				"test-cm",
				errors.New("namespace not found"),
			)
		})

		ctx := context.WithValue(context.Background(), auth.KubeKey, kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.CreateSyncLimitRequest{
			Name:      "test-cm",
			Namespace: "non-existent-ns",
			Key:       "test-key",
			SizeLimit: 100,
		}

		_, err := server.CreateSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.PermissionDenied, statusErr.Code())
		require.Contains(t, statusErr.Message(), "namespace not found")
	})

	t.Run("Create new ConfigMap", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.CreateSyncLimitRequest{
			Name:      "test-cm",
			Namespace: "test-ns",
			Key:       "test-key",
			SizeLimit: 100,
		}

		resp, err := server.CreateSyncLimit(ctx, req)

		require.NoError(t, err)
		require.Equal(t, "test-cm", resp.Name)
		require.Equal(t, "test-ns", resp.Namespace)
		require.Equal(t, "test-key", resp.Key)
		require.Equal(t, int32(100), resp.SizeLimit)
	})

	t.Run("ConfigMap already exists", func(t *testing.T) {
		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{
				"existing-key": "50",
			},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.CreateSyncLimitRequest{
			Name:      "existing-cm",
			Namespace: "test-ns",
			Key:       "new-key",
			SizeLimit: 200,
		}

		resp, err := server.CreateSyncLimit(ctx, req)

		require.NoError(t, err)
		require.Equal(t, "existing-cm", resp.Name)
		require.Equal(t, "test-ns", resp.Namespace)
		require.Equal(t, "new-key", resp.Key)
		require.Equal(t, int32(200), resp.SizeLimit)
	})

	t.Run("ConfigMap exists with nil Data", func(t *testing.T) {

		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nil-data-cm",
				Namespace: "test-ns",
			},
			Data: nil,
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.CreateSyncLimitRequest{
			Name:      "nil-data-cm",
			Namespace: "test-ns",
			Key:       "test-key",
			SizeLimit: 300,
		}

		resp, err := server.CreateSyncLimit(ctx, req)

		require.NoError(t, err)
		require.Equal(t, "nil-data-cm", resp.Name)
		require.Equal(t, "test-key", resp.Key)
		require.Equal(t, int32(300), resp.SizeLimit)
	})
}

func Test_syncServer_GetSyncLimit(t *testing.T) {
	t.Run("ConfigMap doesn't exist", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.GetSyncLimitRequest{
			Name:      "non-existent-cm",
			Namespace: "test-ns",
			Key:       "test-key",
		}

		_, err := server.GetSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, statusErr.Code())
		require.Contains(t, statusErr.Message(), "not found")
	})

	t.Run("Key doesn't exist", func(t *testing.T) {

		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{
				"existing-key": "100",
			},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.GetSyncLimitRequest{
			Name:      "existing-cm",
			Namespace: "test-ns",
			Key:       "non-existent-key",
		}

		_, err := server.GetSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, statusErr.Code())
		require.Contains(t, statusErr.Message(), "key non-existent-key not found")
	})

	t.Run("Invalid size limit format", func(t *testing.T) {

		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{
				"invalid-key": "not-a-number",
			},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.GetSyncLimitRequest{
			Name:      "existing-cm",
			Namespace: "test-ns",
			Key:       "invalid-key",
		}

		_, err := server.GetSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, statusErr.Code())
		require.Contains(t, statusErr.Message(), "invalid size limit format")
	})

	t.Run("Successfully get sync limit", func(t *testing.T) {
		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{
				"valid-key": "500",
			},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.GetSyncLimitRequest{
			Name:      "existing-cm",
			Namespace: "test-ns",
			Key:       "valid-key",
		}

		resp, err := server.GetSyncLimit(ctx, req)

		require.NoError(t, err)
		require.Equal(t, "existing-cm", resp.Name)
		require.Equal(t, "test-ns", resp.Namespace)
		require.Equal(t, "valid-key", resp.Key)
		require.Equal(t, int32(500), resp.SizeLimit)
	})
}

func Test_syncServer_UpdateSyncLimit(t *testing.T) {
	t.Run("SizeLimit <= 0", func(t *testing.T) {
		ctx := context.Background()
		server := NewSyncServer(ctx, fake.NewClientset(), "", nil)

		req := &syncpkg.UpdateSyncLimitRequest{
			Name:      "test-cm",
			Namespace: "test-ns",
			Key:       "test-key",
			SizeLimit: 0,
		}

		_, err := server.UpdateSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, statusErr.Code())
		require.Contains(t, statusErr.Message(), "size limit must be greater than zero")
	})

	t.Run("ConfigMap doesn't exist", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.UpdateSyncLimitRequest{
			Name:      "non-existent-cm",
			Namespace: "test-ns",
			Key:       "test-key",
			SizeLimit: 100,
		}

		_, err := server.UpdateSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, statusErr.Code())
		require.Contains(t, statusErr.Message(), "not found")
	})

	t.Run("ConfigMap with nil Data", func(t *testing.T) {
		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nil-data-cm",
				Namespace: "test-ns",
			},
			Data: nil,
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.UpdateSyncLimitRequest{
			Name:      "nil-data-cm",
			Namespace: "test-ns",
			Key:       "test-key",
			SizeLimit: 200,
		}

		_, err := server.UpdateSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, statusErr.Code())
		require.Contains(t, statusErr.Message(), "please create it first")
	})

	t.Run("Key doesn't exist", func(t *testing.T) {

		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{
				"existing-key": "100",
			},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.UpdateSyncLimitRequest{
			Name:      "existing-cm",
			Namespace: "test-ns",
			Key:       "non-existent-key",
			SizeLimit: 200,
		}

		_, err := server.UpdateSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, statusErr.Code())
		require.Contains(t, statusErr.Message(), "please create it first")
	})

	t.Run("Error updating ConfigMap", func(t *testing.T) {
		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{
				"existing-key": "100",
			},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)

		kubeClient.PrependReactor("update", "configmaps", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("update error")
		})

		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.UpdateSyncLimitRequest{
			Name:      "existing-cm",
			Namespace: "test-ns",
			Key:       "existing-key",
			SizeLimit: 200,
		}

		_, err := server.UpdateSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Internal, statusErr.Code())
		require.Contains(t, statusErr.Message(), "update error")
	})

	t.Run("Successfully update sync limit", func(t *testing.T) {
		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{
				"existing-key": "100",
			},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.UpdateSyncLimitRequest{
			Name:      "existing-cm",
			Namespace: "test-ns",
			Key:       "existing-key",
			SizeLimit: 300,
		}

		resp, err := server.UpdateSyncLimit(ctx, req)

		require.NoError(t, err)
		require.Equal(t, "existing-cm", resp.Name)
		require.Equal(t, "test-ns", resp.Namespace)
		require.Equal(t, "existing-key", resp.Key)
		require.Equal(t, int32(300), resp.SizeLimit)
	})
}

func Test_syncServer_DeleteSyncLimit(t *testing.T) {
	t.Run("ConfigMap doesn't exist", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.DeleteSyncLimitRequest{
			Name:      "non-existent-cm",
			Namespace: "test-ns",
			Key:       "test-key",
		}

		_, err := server.DeleteSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, statusErr.Code())
		require.Contains(t, statusErr.Message(), "not found")
	})

	t.Run("ConfigMap with nil Data", func(t *testing.T) {
		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nil-data-cm",
				Namespace: "test-ns",
			},
			Data: nil,
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.DeleteSyncLimitRequest{
			Name:      "nil-data-cm",
			Namespace: "test-ns",
			Key:       "test-key",
		}

		_, err := server.DeleteSyncLimit(ctx, req)

		require.NoError(t, err)
	})

	t.Run("ConfigMap with empty Data", func(t *testing.T) {
		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "empty-data-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.DeleteSyncLimitRequest{
			Name:      "empty-data-cm",
			Namespace: "test-ns",
			Key:       "test-key",
		}

		_, err := server.DeleteSyncLimit(ctx, req)

		require.NoError(t, err)
	})

	t.Run("Error updating ConfigMap", func(t *testing.T) {
		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{
				"existing-key": "100",
			},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)

		kubeClient.PrependReactor("update", "configmaps", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("update error")
		})

		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.DeleteSyncLimitRequest{
			Name:      "existing-cm",
			Namespace: "test-ns",
			Key:       "existing-key",
		}

		_, err := server.DeleteSyncLimit(ctx, req)

		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Internal, statusErr.Code())
		require.Contains(t, statusErr.Message(), "update error")
	})

	t.Run("Successfully delete sync limit", func(t *testing.T) {
		existingCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-cm",
				Namespace: "test-ns",
			},
			Data: map[string]string{
				"key1": "100",
				"key2": "200",
			},
		}
		kubeClient := fake.NewSimpleClientset(existingCM)
		ctx := withKubeClient(kubeClient)
		server := NewSyncServer(ctx, kubeClient, "", nil)

		req := &syncpkg.DeleteSyncLimitRequest{
			Name:      "existing-cm",
			Namespace: "test-ns",
			Key:       "key1",
		}

		_, err := server.DeleteSyncLimit(ctx, req)

		require.NoError(t, err)
	})
}
