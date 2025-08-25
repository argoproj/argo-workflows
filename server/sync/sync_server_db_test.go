package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/upper/db/v4"

	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	syncdb "github.com/argoproj/argo-workflows/v3/util/sync/db"
	syncdbmocks "github.com/argoproj/argo-workflows/v3/util/sync/db/mocks"
)

func TestDBSyncProvider(t *testing.T) {
	mockSyncQueries := &syncdbmocks.SyncQueries{}
	provider := &dbSyncProvider{db: mockSyncQueries}
	server := &syncServer{
		providers: map[syncpkg.SyncConfigType]SyncConfigProvider{
			syncpkg.SyncConfigType_DATABASE: provider,
		},
	}

	kubeClient := &kubefake.Clientset{}
	allowed := true

	kubeClient.AddReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed},
		}, nil
	})

	kubeClient.AddReactor("create", "selfsubjectrulesreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		var rules []authorizationv1.ResourceRule
		if allowed {
			rules = append(rules, authorizationv1.ResourceRule{})
		}
		return true, &authorizationv1.SelfSubjectRulesReview{
			Status: authorizationv1.SubjectRulesReviewStatus{
				ResourceRules: rules,
			},
		}, nil
	})

	ctx := context.WithValue(logging.TestContext(t.Context()), auth.KubeKey, kubeClient)

	t.Run("CreateSyncLimit", func(t *testing.T) {
		req := &syncpkg.CreateSyncLimitRequest{
			Type:      syncpkg.SyncConfigType_DATABASE,
			Namespace: "test-ns",
			Key:       "test-name",
			SizeLimit: 5,
		}

		allowed = false
		resp, err := provider.createSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.PermissionDenied, status.Code(err))

		allowed = true

		mockSyncQueries.On("GetSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(nil, assert.AnError).Once()
		resp, err = provider.createSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.Internal, status.Code(err))

		mockSyncQueries.On("GetSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(&syncdb.LimitRecord{}, nil).Once()
		resp, err = provider.createSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.AlreadyExists, status.Code(err))

		mockSyncQueries.On("GetSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(nil, db.ErrNoMoreRows).Once()
		mockSyncQueries.On("CreateSemaphoreLimit", mock.Anything, "test-ns/test-name", 5).Return(assert.AnError).Once()
		resp, err = provider.createSyncLimit(ctx, req)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.Internal, status.Code(err))

		mockSyncQueries.On("GetSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(nil, db.ErrNoMoreRows).Once()
		mockSyncQueries.On("CreateSemaphoreLimit", mock.Anything, "test-ns/test-name", 5).Return(nil).Once()

		resp, err = server.CreateSyncLimit(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "test-name", resp.Name)
		require.Equal(t, "test-ns", resp.Namespace)
		require.Equal(t, int32(5), resp.SizeLimit)
	})

	t.Run("GetSyncLimit", func(t *testing.T) {
		req := &syncpkg.GetSyncLimitRequest{
			Type:      syncpkg.SyncConfigType_DATABASE,
			Namespace: "test-ns",
			Key:       "test-name",
		}

		allowed = false
		resp, err := provider.getSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.PermissionDenied, status.Code(err))

		allowed = true

		mockSyncQueries.On("GetSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(nil, db.ErrNoMoreRows).Once()
		resp, err = provider.getSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.NotFound, status.Code(err))

		mockSyncQueries.On("GetSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(nil, assert.AnError).Once()
		resp, err = provider.getSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.Internal, status.Code(err))

		mockSyncQueries.On("GetSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(&syncdb.LimitRecord{
			SizeLimit: 5,
		}, nil).Once()

		resp, err = provider.getSyncLimit(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "test-name", resp.Name)
		require.Equal(t, "test-ns", resp.Namespace)
		require.Equal(t, int32(5), resp.SizeLimit)
	})

	t.Run("UpdateSyncLimit", func(t *testing.T) {
		allowed = false

		req := &syncpkg.UpdateSyncLimitRequest{
			Type:      syncpkg.SyncConfigType_DATABASE,
			Namespace: "test-ns",
			Key:       "test-name",
			SizeLimit: 10,
		}
		resp, err := provider.updateSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.PermissionDenied, status.Code(err))

		allowed = true

		mockSyncQueries.On("UpdateSemaphoreLimit", mock.Anything, "test-ns/test-name", 10).Return(db.ErrNoMoreRows).Once()
		resp, err = provider.updateSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.NotFound, status.Code(err))

		mockSyncQueries.On("UpdateSemaphoreLimit", mock.Anything, "test-ns/test-name", 10).Return(assert.AnError).Once()
		resp, err = provider.updateSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.Internal, status.Code(err))

		mockSyncQueries.On("UpdateSemaphoreLimit", mock.Anything, "test-ns/test-name", 10).Return(nil).Once()
		resp, err = provider.updateSyncLimit(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "test-name", resp.Name)
		require.Equal(t, "test-ns", resp.Namespace)
		require.Equal(t, int32(10), resp.SizeLimit)
	})

	t.Run("DeleteSyncLimit", func(t *testing.T) {
		allowed = false

		req := &syncpkg.DeleteSyncLimitRequest{
			Type:      syncpkg.SyncConfigType_DATABASE,
			Namespace: "test-ns",
			Key:       "test-name",
		}
		resp, err := provider.deleteSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.PermissionDenied, status.Code(err))

		allowed = true

		mockSyncQueries.On("DeleteSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(db.ErrNoMoreRows).Once()
		resp, err = provider.deleteSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.NotFound, status.Code(err))

		mockSyncQueries.On("DeleteSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(assert.AnError).Once()
		resp, err = provider.deleteSyncLimit(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, codes.Internal, status.Code(err))

		mockSyncQueries.On("DeleteSemaphoreLimit", mock.Anything, "test-ns/test-name").Return(nil).Once()
		resp, err = provider.deleteSyncLimit(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}
