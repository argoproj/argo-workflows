package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	utilsqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

func TestUpdateConfig(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	err := controller.updateConfig(ctx)
	require.NoError(t, err)
	assert.NotNil(t, controller.Config)
	assert.NotNil(t, controller.archiveLabelSelector)
	assert.NotNil(t, controller.wfArchive)
	assert.NotNil(t, controller.offloadNodeStatusRepo)
}

func TestGetMemoizationCacheRetriesBackendInitialization(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.memoLock.Lock()
	controller.memoConfig = &config.MemoizationConfig{
		DBConfig: config.DBConfig{
			PostgreSQL: &config.PostgreSQLConfig{},
		},
	}
	controller.memoLock.Unlock()

	originalBuilder := memoSessionProxyFromConfig
	originalMigrate := memoizationMigrate
	t.Cleanup(func() {
		memoSessionProxyFromConfig = originalBuilder
		memoizationMigrate = originalMigrate
	})

	calls := 0
	memoSessionProxyFromConfig = func(context.Context, kubernetes.Interface, string, *config.MemoizationConfig) *utilsqldb.SessionProxy {
		calls++
		if calls == 1 {
			return nil
		}
		return utilsqldb.NewSessionProxyFromSession(nil, &config.DBConfig{PostgreSQL: &config.PostgreSQLConfig{}}, "", "")
	}
	memoizationMigrate = func(context.Context, *utilsqldb.SessionProxy, memodb.Config) error {
		return nil
	}

	cache := controller.getMemoizationCache(ctx, "default", "memo-cache")
	assert.Nil(t, cache)

	cache = controller.getMemoizationCache(ctx, "default", "memo-cache")
	require.NotNil(t, cache)
	assert.Equal(t, 2, calls)
}
