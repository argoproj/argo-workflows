package controller

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
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
	assert.Equal(t, memodb.NullMemoizationDB, controller.memoQueries)
}

func TestUpdateConfigMemoizationSessionFailureDisablesMemoization(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Memoization = &config.MemoizationConfig{}

	origMemoSessionProxyFromConfig := memoSessionProxyFromConfig
	memoSessionProxyFromConfig = func(context.Context, kubernetes.Interface, string, *config.MemoizationConfig) *sqldb.SessionProxy {
		return nil
	}
	t.Cleanup(func() {
		memoSessionProxyFromConfig = origMemoSessionProxyFromConfig
	})

	err := controller.updateConfig(ctx)
	require.NoError(t, err)
	assert.Nil(t, controller.memoSessionProxy)
	assert.Equal(t, memodb.NullMemoizationDB, controller.getMemoizationQueries())

	cache := controller.getMemoizationCache(ctx, "default", "memo-disabled-cache")
	require.NotNil(t, cache)
	require.NoError(t, cache.Save(ctx, "memo-key", "", &wfv1.Outputs{}, ""))

	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "memo-disabled-cache", metav1.GetOptions{})
	assert.True(t, apierr.IsNotFound(err))
}

type observingCacheFactory struct {
	setQueries func(memodb.MemoizationDB)
}

func (o *observingCacheFactory) GetCache(context.Context, controllercache.Type, string, string) controllercache.MemoizationCache {
	return nil
}

func (o *observingCacheFactory) SetQueries(q memodb.MemoizationDB) {
	if o.setQueries != nil {
		o.setQueries(q)
	}
}

func TestUpdateConfigMemoizationDisableDetachesCachesBeforeClosingSession(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	sessionProxy := sqldb.NewSessionProxyFromSession(nil, &config.DBConfig{}, "user", "password")
	controller.memoSessionProxy = sessionProxy

	var setQueriesSawClosedProxy bool
	controller.cacheFactory = &observingCacheFactory{
		setQueries: func(memodb.MemoizationDB) {
			err := sessionProxy.With(ctx, func(db.Session) error { return nil })
			setQueriesSawClosedProxy = err != nil && err.Error() == "session proxy is closed"
		},
	}

	err := controller.updateConfig(ctx)
	require.NoError(t, err)
	assert.False(t, setQueriesSawClosedProxy)
	assert.Nil(t, controller.memoSessionProxy)
	require.EqualError(t, sessionProxy.With(ctx, func(db.Session) error { return nil }), "session proxy is closed")
}

func TestUpdateConfigMemoizationMigrationFailureDisablesMemoization(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Memoization = &config.MemoizationConfig{}
	sessionProxy := sqldb.NewSessionProxyFromSession(nil, &config.DBConfig{}, "user", "password")
	controller.memoSessionProxy = sessionProxy

	origMemoizationMigrate := memoizationMigrate
	memoizationMigrate = func(context.Context, *sqldb.SessionProxy, memodb.Config) error {
		return stderrors.New("boom")
	}
	t.Cleanup(func() {
		memoizationMigrate = origMemoizationMigrate
	})

	err := controller.updateConfig(ctx)
	require.NoError(t, err)
	assert.Nil(t, controller.memoSessionProxy)
	assert.Equal(t, memodb.NullMemoizationDB, controller.getMemoizationQueries())
	require.EqualError(t, sessionProxy.With(ctx, func(db.Session) error { return nil }), "session proxy is closed")

	cache := controller.getMemoizationCache(ctx, "default", "memo-migrate-disabled-cache")
	require.NotNil(t, cache)
	require.NoError(t, cache.Save(ctx, "memo-key", "", &wfv1.Outputs{}, ""))

	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "memo-migrate-disabled-cache", metav1.GetOptions{})
	assert.True(t, apierr.IsNotFound(err))
}
