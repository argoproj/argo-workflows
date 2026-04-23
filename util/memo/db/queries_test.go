//go:build !windows

package db_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const (
	testDBName     = "memotest"
	testDBUser     = "user"
	testDBPassword = "pass"
	testTableName  = "memoization_cache"
	testNamespace  = "default"
	testCacheName  = "my-cache"
)

// setupPostgres starts a throwaway Postgres container and returns a migrated SessionProxy.
func setupPostgres(ctx context.Context, t *testing.T) *sqldb.SessionProxy {
	t.Helper()
	pg, err := testpostgres.Run(ctx,
		"postgres:17.4-alpine",
		testpostgres.WithDatabase(testDBName),
		testpostgres.WithUsername(testDBUser),
		testpostgres.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(pg); termErr != nil {
			t.Logf("failed to terminate postgres container: %s", termErr)
		}
	})

	host, err := pg.Host(ctx)
	require.NoError(t, err)
	portStr, err := pg.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr.Port())
	require.NoError(t, err)

	dbCfg := config.DBConfig{
		PostgreSQL: &config.PostgreSQLConfig{
			DatabaseConfig: config.DatabaseConfig{
				Host:     host,
				Port:     port,
				Database: testDBName,
			},
		},
	}
	sp, err := sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{
		DBConfig:   dbCfg,
		Username:   testDBUser,
		Password:   testDBPassword,
		MaxRetries: 5,
		BaseDelay:  200 * time.Millisecond,
		MaxDelay:   10 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sp.Close() })

	memoCfg := &config.MemoizationConfig{
		DBConfig:  dbCfg,
		TableName: testTableName,
	}
	require.NoError(t, memodb.Migrate(ctx, sp, memodb.ConfigFromConfig(memoCfg)))
	return sp
}

// setupMySQL starts a throwaway MySQL container and returns a migrated SessionProxy.
func setupMySQL(ctx context.Context, t *testing.T) *sqldb.SessionProxy {
	t.Helper()
	my, err := testmysql.Run(ctx,
		"mysql:8.4.5",
		testmysql.WithDatabase(testDBName),
		testmysql.WithUsername(testDBUser),
		testmysql.WithPassword(testDBPassword),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(my); termErr != nil {
			t.Logf("failed to terminate mysql container: %s", termErr)
		}
	})

	host, err := my.Host(ctx)
	require.NoError(t, err)
	portStr, err := my.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr.Port())
	require.NoError(t, err)

	dbCfg := config.DBConfig{
		MySQL: &config.MySQLConfig{
			DatabaseConfig: config.DatabaseConfig{
				Host:     host,
				Port:     port,
				Database: testDBName,
			},
		},
	}
	sp, err := sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{
		DBConfig:   dbCfg,
		Username:   testDBUser,
		Password:   testDBPassword,
		MaxRetries: 5,
		BaseDelay:  200 * time.Millisecond,
		MaxDelay:   10 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sp.Close() })

	memoCfg := &config.MemoizationConfig{
		DBConfig:  dbCfg,
		TableName: testTableName,
	}
	require.NoError(t, memodb.Migrate(ctx, sp, memodb.ConfigFromConfig(memoCfg)))
	return sp
}

func sampleOutputs(message string) *wfv1.Outputs {
	return &wfv1.Outputs{
		Parameters: []wfv1.Parameter{
			{Name: "result", Value: wfv1.AnyStringPtr(message)},
		},
	}
}

func TestQueriesSaveAndLoad(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupPostgres(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.Postgres)
	require.NoError(t, err)

	// Load returns nil when no entry exists.
	rec, err := q.Load(ctx, sp, testNamespace, testCacheName, "key1")
	require.NoError(t, err)
	assert.Nil(t, rec, "expected nil for missing key")

	// Save an entry and load it back.
	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "key1", "node-abc", sampleOutputs("hello"), 2592000))
	rec, err = q.Load(ctx, sp, testNamespace, testCacheName, "key1")
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "node-abc", rec.NodeID)
	assert.Contains(t, rec.Outputs, "hello")
}

func TestQueriesNamespaceIsolation(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupPostgres(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.Postgres)
	require.NoError(t, err)

	// Save the same cache_name+cache key in two different namespaces.
	require.NoError(t, q.Save(ctx, sp, "ns-a", testCacheName, "shared-key", "node-a", sampleOutputs("from-a"), 2592000))
	require.NoError(t, q.Save(ctx, sp, "ns-b", testCacheName, "shared-key", "node-b", sampleOutputs("from-b"), 2592000))

	// Each namespace should see its own entry.
	recA, err := q.Load(ctx, sp, "ns-a", testCacheName, "shared-key")
	require.NoError(t, err)
	require.NotNil(t, recA)
	assert.Equal(t, "node-a", recA.NodeID)
	assert.Contains(t, recA.Outputs, "from-a")

	recB, err := q.Load(ctx, sp, "ns-b", testCacheName, "shared-key")
	require.NoError(t, err)
	require.NotNil(t, recB)
	assert.Equal(t, "node-b", recB.NodeID)
	assert.Contains(t, recB.Outputs, "from-b")
}

func TestQueriesSaveReplaces(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupPostgres(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.Postgres)
	require.NoError(t, err)

	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "key3", "node-old", sampleOutputs("old"), 2592000))
	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "key3", "node-new", sampleOutputs("new"), 2592000))

	rec, err := q.Load(ctx, sp, testNamespace, testCacheName, "key3")
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "node-new", rec.NodeID)
	assert.Contains(t, rec.Outputs, "new")
}

func TestQueriesLoadSkipsExpiredEntries(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupPostgres(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.Postgres)
	require.NoError(t, err)

	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "expired-key", "node-old", sampleOutputs("old"), 2592000))

	_, err = sp.Session().SQL().
		ExecContext(ctx, `UPDATE `+testTableName+` SET expires_at = $1 WHERE cache_key = $2`, time.Now().Add(-10*time.Second), "expired-key")
	require.NoError(t, err)

	rec, err := q.Load(ctx, sp, testNamespace, testCacheName, "expired-key")
	require.NoError(t, err)
	assert.Nil(t, rec, "expired entries should load as a cache miss")
}

func TestQueriesPruneRemovesOldEntries(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupPostgres(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.Postgres)
	require.NoError(t, err)

	// Save an entry with a very short max_age (1 second) and one with 30 days.
	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "old-key", "node-old", sampleOutputs("old"), 1))
	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "new-key", "node-new", sampleOutputs("new"), 2592000))

	// Backdate old-key's expires_at so it is in the past.
	_, err = sp.Session().SQL().
		ExecContext(ctx, `UPDATE `+testTableName+` SET expires_at = $1 WHERE cache_key = $2`, time.Now().Add(-10*time.Second), "old-key")
	require.NoError(t, err)

	// Prune — old-key should be deleted (expires_at < now), new-key should survive.
	n, err := q.Prune(ctx, sp)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "expected exactly one row pruned")

	old, err := q.Load(ctx, sp, testNamespace, testCacheName, "old-key")
	require.NoError(t, err)
	assert.Nil(t, old, "old entry should have been pruned")

	fresh, err := q.Load(ctx, sp, testNamespace, testCacheName, "new-key")
	require.NoError(t, err)
	assert.NotNil(t, fresh, "new entry should still exist")
}

func TestQueriesPruneKeepsRecentEntries(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupPostgres(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.Postgres)
	require.NoError(t, err)

	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "recent", "node-1", sampleOutputs("v1"), 2592000))

	// All entries are recent — nothing should be pruned.
	n, err := q.Prune(ctx, sp)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n, "expected no rows pruned when all entries are fresh")
}

// MySQL test variants — verify longtext and ON DUPLICATE KEY UPDATE.

func TestMySQLSaveAndLoad(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupMySQL(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.MySQL)
	require.NoError(t, err)

	rec, err := q.Load(ctx, sp, testNamespace, testCacheName, "key1")
	require.NoError(t, err)
	assert.Nil(t, rec, "expected nil for missing key")

	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "key1", "node-abc", sampleOutputs("hello"), 2592000))
	rec, err = q.Load(ctx, sp, testNamespace, testCacheName, "key1")
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "node-abc", rec.NodeID)
	assert.Contains(t, rec.Outputs, "hello")
}

func TestMySQLSaveReplaces(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupMySQL(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.MySQL)
	require.NoError(t, err)

	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "key3", "node-old", sampleOutputs("old"), 2592000))
	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "key3", "node-new", sampleOutputs("new"), 2592000))

	rec, err := q.Load(ctx, sp, testNamespace, testCacheName, "key3")
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "node-new", rec.NodeID)
	assert.Contains(t, rec.Outputs, "new")
}

func TestMySQLLoadSkipsExpiredEntries(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupMySQL(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.MySQL)
	require.NoError(t, err)

	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "expired-key", "node-old", sampleOutputs("old"), 2592000))

	_, err = sp.Session().SQL().
		ExecContext(ctx, "UPDATE "+testTableName+" SET expires_at = ? WHERE cache_key = ?", time.Now().Add(-10*time.Second), "expired-key")
	require.NoError(t, err)

	rec, err := q.Load(ctx, sp, testNamespace, testCacheName, "expired-key")
	require.NoError(t, err)
	assert.Nil(t, rec, "expired entries should load as a cache miss")
}

func TestMySQLPruneRemovesOldEntries(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupMySQL(ctx, t)
	q, err := memodb.NewQueries(testTableName, sqldb.MySQL)
	require.NoError(t, err)

	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "old-key", "node-old", sampleOutputs("old"), 1))
	require.NoError(t, q.Save(ctx, sp, testNamespace, testCacheName, "new-key", "node-new", sampleOutputs("new"), 2592000))

	// Backdate old-key's expires_at so it is in the past.
	_, err = sp.Session().SQL().
		ExecContext(ctx, "UPDATE "+testTableName+" SET expires_at = ? WHERE cache_key = ?", time.Now().Add(-10*time.Second), "old-key")
	require.NoError(t, err)

	n, err := q.Prune(ctx, sp)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "expected exactly one row pruned")

	old, err := q.Load(ctx, sp, testNamespace, testCacheName, "old-key")
	require.NoError(t, err)
	assert.Nil(t, old, "old entry should have been pruned")

	fresh, err := q.Load(ctx, sp, testNamespace, testCacheName, "new-key")
	require.NoError(t, err)
	assert.NotNil(t, fresh, "new entry should still exist")
}
