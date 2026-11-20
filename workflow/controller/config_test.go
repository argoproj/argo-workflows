package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

func memoBackend(t *testing.T, c *WorkflowController) memodb.MemoizationDB {
	t.Helper()
	var result memodb.MemoizationDB
	require.NoError(t, c.cacheFactory.WithQueries(func(q memodb.MemoizationDB) error { result = q; return nil }))
	return result
}

func TestUpdateConfig(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, c := newController(ctx)
	defer cancel()
	require.NoError(t, c.updateConfig(ctx))
	assert.NotNil(t, c.Config)
	assert.NotNil(t, c.archiveLabelSelector)
	assert.NotNil(t, c.wfArchive)
	assert.NotNil(t, c.offloadNodeStatusRepo)
	assert.False(t, memoBackend(t, c).IsEnabled())
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

func TestMemoizationOnlyInitializesOnLeader(t *testing.T) {
	for _, skipPersistence := range []bool{false, true} {
		t.Run(map[bool]string{false: "no-persistence", true: "skip-persistence-migration"}[skipPersistence], func(t *testing.T) {
			stubMemoizationSetup(t)
			ctx := logging.TestContext(t.Context())
			cancel, c := newController(ctx)
			defer cancel()
			c.Config.Memoization = &config.MemoizationConfig{}
			connects, migrations := 0, 0
			connect := memoSessionProxyFromConfig
			memoSessionProxyFromConfig = func(ctx context.Context, k kubernetes.Interface, ns string, cfg *config.MemoizationConfig) (*sqldb.SessionProxy, error) {
				connects++
				return connect(ctx, k, ns, cfg)
			}
			memoizationMigrate = func(context.Context, *sqldb.SessionProxy, memodb.Config) error { migrations++; return nil }
			require.NoError(t, c.updateConfig(ctx))
			assert.Zero(t, connects, "standbys must not open memoization pools")
			assert.Zero(t, migrations, "standbys must not run migrations")
			if skipPersistence {
				c.Config.Persistence = &config.PersistConfig{SkipMigration: true}
			}
			require.NoError(t, c.initDB(ctx))
			assert.Equal(t, 1, connects)
			assert.Equal(t, 1, migrations)
			assert.True(t, memoBackend(t, c).IsEnabled())
			c.cacheFactory.SetQueries(ctx, nil)
		})
	}
}

func TestMemoizationUnchangedConfigDoesNotReconnect(t *testing.T) {
	stubMemoizationSetup(t)
	ctx := logging.TestContext(t.Context())
	cancel, c := newController(ctx)
	defer cancel()
	c.Config.Memoization = &config.MemoizationConfig{}
	require.NoError(t, c.initDB(ctx))
	original := memoBackend(t, c)
	// Even a database outage must not affect an unrelated configuration edit.
	memoSessionProxyFromConfig = func(context.Context, kubernetes.Interface, string, *config.MemoizationConfig) (*sqldb.SessionProxy, error) {
		t.Fatal("unexpected reconnect")
		return nil, errors.New("offline")
	}
	memoizationMigrate = func(context.Context, *sqldb.SessionProxy, memodb.Config) error {
		t.Fatal("unexpected migration")
		return nil
	}
	c.Config.Parallelism = 42
	c.Config.Memoization = &config.MemoizationConfig{}
	require.NoError(t, c.updateConfig(ctx))
	assert.Same(t, original, memoBackend(t, c))
	c.cacheFactory.SetQueries(ctx, nil)
}

func TestMemoizationInitializationErrors(t *testing.T) {
	for _, stage := range []string{"connect", "migrate", "queries"} {
		t.Run(stage, func(t *testing.T) {
			stubMemoizationSetup(t)
			ctx := logging.TestContext(t.Context())
			cancel, c := newController(ctx)
			defer cancel()
			c.Config.Memoization = &config.MemoizationConfig{}
			candidate := sqldb.NewSessionProxyFromSession(nil, &config.DBConfig{}, "user", "password")
			memoSessionProxyFromConfig = func(context.Context, kubernetes.Interface, string, *config.MemoizationConfig) (*sqldb.SessionProxy, error) {
				if stage == "connect" {
					return nil, errors.New("offline")
				}
				return candidate, nil
			}
			if stage == "migrate" {
				memoizationMigrate = func(context.Context, *sqldb.SessionProxy, memodb.Config) error { return errors.New("migration failed") }
			}
			if stage == "queries" {
				c.Config.Memoization.TableName = "invalid-name"
			}
			require.Error(t, c.initDB(ctx))
			assert.False(t, c.memoizationInitialized)
			assert.False(t, memoBackend(t, c).IsEnabled())
			if stage != "connect" {
				require.EqualError(t, candidate.With(ctx, func(db.Session) error { return nil }), "session proxy is closed")
			}
		})
	}
}

func TestMemoizationChangedConfigReplacesBackend(t *testing.T) {
	stubMemoizationSetup(t)
	ctx := logging.TestContext(t.Context())
	cancel, c := newController(ctx)
	defer cancel()
	c.Config.Memoization = &config.MemoizationConfig{}
	require.NoError(t, c.initDB(ctx))
	original := memoBackend(t, c)
	c.Config.Memoization.TableName = "replacement_cache"
	c.Config.Memoization.PostgreSQL = &config.PostgreSQLConfig{DatabaseConfig: config.DatabaseConfig{Host: "new-host"}}
	connect := memoSessionProxyFromConfig
	memoSessionProxyFromConfig = func(ctx context.Context, k kubernetes.Interface, ns string, cfg *config.MemoizationConfig) (*sqldb.SessionProxy, error) {
		assert.Equal(t, "new-host", cfg.PostgreSQL.Host)
		return connect(ctx, k, ns, cfg)
	}
	require.NoError(t, c.updateConfig(ctx))
	current := memoBackend(t, c)
	assert.NotSame(t, original, current)
	c.Config.Memoization.TableName = "failed_replacement"
	memoizationMigrate = func(context.Context, *sqldb.SessionProxy, memodb.Config) error { return errors.New("boom") }
	require.Error(t, c.updateConfig(ctx))
	assert.Same(t, current, memoBackend(t, c))
	c.Config.Memoization = nil
	require.NoError(t, c.updateConfig(ctx))
	assert.False(t, memoBackend(t, c).IsEnabled())
	require.NoError(t, c.cacheFactory.GetCache("after-disable").Save(ctx, "key", "node", nil, ""))
}

type closeErrorSession struct {
	db.Session
	closed chan struct{}
}

func (s *closeErrorSession) Close() error { close(s.closed); return errors.New("close failed") }

func TestMemoizationCloseFailureStillDisablesBackend(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, c := newController(ctx)
	defer cancel()
	session := &closeErrorSession{closed: make(chan struct{})}
	previous := sqldb.NewSessionProxyFromSession(session, &config.DBConfig{}, "user", "password")
	q, err := memodb.NewQueries("cache_entries", previous)
	require.NoError(t, err)
	c.cacheFactory.SetQueries(ctx, q)
	require.NoError(t, c.initDB(ctx))
	select {
	case <-session.closed:
	case <-time.After(time.Second):
		t.Fatal("retired session was not closed")
	}
	assert.False(t, memoBackend(t, c).IsEnabled())
	require.NoError(t, c.cacheFactory.GetCache("after-disable").Save(ctx, "key", "node", nil, ""))
}
