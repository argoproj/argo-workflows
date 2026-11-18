package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
)

func TestUpdateConfig(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	require.NoError(t, controller.updateConfig(ctx))
	assert.NotNil(t, controller.Config)
	assert.NotNil(t, controller.archiveLabelSelector)
	assert.NotNil(t, controller.wfArchive)
	assert.NotNil(t, controller.offloadNodeStatusRepo)
	assert.Equal(t, memodb.NullMemoizationDB, controller.getMemoizationQueries())
}

func stubMemoizationSetup(t *testing.T) {
	t.Helper()
	connect, migrate := memoSessionProxyFromConfig, memoizationMigrate
	t.Cleanup(func() { memoSessionProxyFromConfig, memoizationMigrate = connect, migrate })
	memoSessionProxyFromConfig = func(context.Context, kubernetes.Interface, string, *config.MemoizationConfig) (*sqldb.SessionProxy, error) {
		return sqldb.NewSessionProxyFromSession(nil, &config.DBConfig{}, "user", "password"), nil
	}
	memoizationMigrate = func(context.Context, *sqldb.SessionProxy, memodb.Config) error { return nil }
}

func TestUpdateConfigMemoizationConnectionFailure(t *testing.T) {
	stubMemoizationSetup(t)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.Memoization = &config.MemoizationConfig{}
	connectionErr := errors.New("connection refused")
	memoSessionProxyFromConfig = func(context.Context, kubernetes.Interface, string, *config.MemoizationConfig) (*sqldb.SessionProxy, error) {
		return nil, connectionErr
	}
	require.ErrorIs(t, controller.updateConfig(ctx), connectionErr)
	assert.Nil(t, controller.memoSessionProxy)
	assert.False(t, controller.getMemoizationQueries().IsEnabled())
}

func TestUpdateConfigMemoizationMigrationFailure(t *testing.T) {
	stubMemoizationSetup(t)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.Memoization = &config.MemoizationConfig{}
	var failedSession *sqldb.SessionProxy
	migrationErr := errors.New("migration failed")
	memoizationMigrate = func(_ context.Context, sp *sqldb.SessionProxy, _ memodb.Config) error {
		failedSession = sp
		return migrationErr
	}
	require.ErrorIs(t, controller.updateConfig(ctx), migrationErr)
	assert.Nil(t, controller.memoSessionProxy)
	require.EqualError(t, failedSession.With(ctx, func(db.Session) error { return nil }), "session proxy is closed")
}

func TestUpdateConfigMemoizationInvalidTableClosesNewSession(t *testing.T) {
	stubMemoizationSetup(t)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.Memoization = &config.MemoizationConfig{TableName: "invalid-name", SkipMigration: true}
	candidate := sqldb.NewSessionProxyFromSession(nil, &config.DBConfig{}, "user", "password")
	memoSessionProxyFromConfig = func(context.Context, kubernetes.Interface, string, *config.MemoizationConfig) (*sqldb.SessionProxy, error) {
		return candidate, nil
	}
	require.ErrorContains(t, controller.updateConfig(ctx), "invalid table name")
	assert.Nil(t, controller.memoSessionProxy)
	assert.False(t, controller.getMemoizationQueries().IsEnabled())
	require.EqualError(t, candidate.With(ctx, func(db.Session) error { return nil }), "session proxy is closed")
}

type closeErrorSession struct {
	db.Session
	closed bool
}

func (s *closeErrorSession) Close() error {
	s.closed = true
	return errors.New("close failed")
}

func TestUpdateConfigMemoizationCloseFailureStillDisablesBackend(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	session := &closeErrorSession{}
	previous := sqldb.NewSessionProxyFromSession(session, &config.DBConfig{}, "user", "password")
	controller.memoSessionProxy = previous
	queries, err := memodb.NewQueries("cache_entries", previous)
	require.NoError(t, err)
	controller.setMemoizationQueries(queries)
	controller.cacheFactory.SetQueries(queries)
	require.NoError(t, controller.updateConfig(ctx))
	assert.True(t, session.closed)
	assert.Nil(t, controller.memoSessionProxy)
	assert.False(t, controller.getMemoizationQueries().IsEnabled())
	require.NoError(t, controller.cacheFactory.GetCache("after-disable").Save(ctx, "key", "node", nil, ""))
}

func TestUpdateConfigMemoizationRecreatesConnection(t *testing.T) {
	stubMemoizationSetup(t)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.Memoization = &config.MemoizationConfig{}
	require.NoError(t, controller.updateConfig(ctx))
	oldSession := controller.memoSessionProxy
	oldQueries := controller.getMemoizationQueries()
	t.Cleanup(func() { _ = controller.memoSessionProxy.Close() })

	controller.Config.Memoization.TableName = "replacement_cache"
	controller.Config.Memoization.PostgreSQL = &config.PostgreSQLConfig{DatabaseConfig: config.DatabaseConfig{Host: "new-host"}}
	connect := memoSessionProxyFromConfig
	memoSessionProxyFromConfig = func(ctx context.Context, client kubernetes.Interface, ns string, cfg *config.MemoizationConfig) (*sqldb.SessionProxy, error) {
		assert.Equal(t, "new-host", cfg.PostgreSQL.Host)
		return connect(ctx, client, ns, cfg)
	}
	require.NoError(t, controller.updateConfig(ctx))
	assert.NotSame(t, oldSession, controller.memoSessionProxy)
	assert.NotSame(t, oldQueries, controller.getMemoizationQueries())
	require.EqualError(t, oldSession.With(ctx, func(db.Session) error { return nil }), "session proxy is closed")

	// A failed reload must not replace the working backend.
	currentSession, currentQueries := controller.memoSessionProxy, controller.getMemoizationQueries()
	memoizationMigrate = func(context.Context, *sqldb.SessionProxy, memodb.Config) error { return errors.New("boom") }
	require.Error(t, controller.updateConfig(ctx))
	assert.Same(t, currentSession, controller.memoSessionProxy)
	assert.Same(t, currentQueries, controller.getMemoizationQueries())
}

type observingCacheFactory struct{ setQueries func(memodb.MemoizationDB) }

func (*observingCacheFactory) GetCache(string) controllercache.MemoizationCache { return nil }
func (o *observingCacheFactory) SetQueries(q memodb.MemoizationDB)              { o.setQueries(q) }

func TestUpdateConfigMemoizationDisableDetachesCachesBeforeClosingSession(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	session := sqldb.NewSessionProxyFromSession(nil, &config.DBConfig{}, "user", "password")
	controller.memoSessionProxy = session
	controller.cacheFactory = &observingCacheFactory{setQueries: func(q memodb.MemoizationDB) {
		assert.Nil(t, q)
		require.EqualError(t, session.With(ctx, func(db.Session) error { return nil }), "no active session")
	}}
	require.NoError(t, controller.updateConfig(ctx))
	assert.Nil(t, controller.memoSessionProxy)
	require.EqualError(t, session.With(ctx, func(db.Session) error { return nil }), "session proxy is closed")
}
