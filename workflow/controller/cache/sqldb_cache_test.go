//go:build !windows

package cache

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const (
	testDBName     = "cachetest"
	testDBUser     = "user"
	testDBPassword = "pass"
	testNamespace  = "default"
	testCacheName  = "my-cache"
)

var testTableName = memodb.TableName(nil)

func setupTestPostgres(ctx context.Context, t *testing.T) *sqldb.SessionProxy {
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
		DBConfig: dbCfg,
	}
	require.NoError(t, memodb.Migrate(ctx, sp, memodb.ConfigFromConfig(memoCfg)))
	return sp
}

func newTestSQLDBCache(t *testing.T, sp *sqldb.SessionProxy) MemoizationCache {
	t.Helper()
	queries, err := memodb.NewQueries(testTableName, sp)
	require.NoError(t, err)
	return newSQLDBCache(testNamespace, testCacheName, queries)
}

func TestSQLDBCacheSaveAndLoad(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupTestPostgres(ctx, t)

	c := newTestSQLDBCache(t, sp)

	// Load returns nil for missing key.
	entry, err := c.Load(ctx, "key1")
	require.NoError(t, err)
	assert.Nil(t, entry)

	// Save and load back.
	outputs := &wfv1.Outputs{
		Parameters: []wfv1.Parameter{
			{Name: "result", Value: wfv1.AnyStringPtr("hello")},
		},
	}
	require.NoError(t, c.Save(ctx, "key1", "node-abc", outputs, "720h"))

	entry, err = c.Load(ctx, "key1")
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, "node-abc", entry.NodeID)
	assert.True(t, entry.Hit())
	require.NotNil(t, entry.Outputs)
	require.Len(t, entry.Outputs.Parameters, 1)
	assert.Equal(t, "result", entry.Outputs.Parameters[0].Name)
	assert.Equal(t, "hello", entry.Outputs.Parameters[0].Value.String())
	assert.False(t, entry.CreationTimestamp.IsZero())
}

func TestSQLDBCacheInvalidKey(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupTestPostgres(ctx, t)

	c := newTestSQLDBCache(t, sp)

	// Keys with invalid characters should be rejected.
	_, err := c.Load(ctx, "invalid key!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cache key")

	err = c.Save(ctx, "invalid key!", "node-1", &wfv1.Outputs{}, "1h")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cache key")
}

func TestSQLDBCacheOutputsRoundTrip(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupTestPostgres(ctx, t)

	c := newTestSQLDBCache(t, sp)

	// Save complex outputs and verify they round-trip through JSON.
	outputs := &wfv1.Outputs{
		Parameters: []wfv1.Parameter{
			{Name: "p1", Value: wfv1.AnyStringPtr("v1")},
			{Name: "p2", Value: wfv1.AnyStringPtr("v2")},
		},
	}
	require.NoError(t, c.Save(ctx, "complex-key", "node-x", outputs, "1h"))

	entry, err := c.Load(ctx, "complex-key")
	require.NoError(t, err)
	require.NotNil(t, entry)

	// Verify the round-tripped outputs match by comparing JSON.
	originalJSON, _ := json.Marshal(outputs)
	loadedJSON, _ := json.Marshal(entry.Outputs)
	assert.JSONEq(t, string(originalJSON), string(loadedJSON))
}

func TestSQLDBCacheGetOutputsWithMaxAge(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupTestPostgres(ctx, t)

	c := newTestSQLDBCache(t, sp)

	outputs := &wfv1.Outputs{
		Parameters: []wfv1.Parameter{
			{Name: "result", Value: wfv1.AnyStringPtr("cached")},
		},
	}
	require.NoError(t, c.Save(ctx, "ttl-key", "node-ttl", outputs, "1h"))

	entry, err := c.Load(ctx, "ttl-key")
	require.NoError(t, err)
	require.NotNil(t, entry)

	// With a large maxAge, outputs should be returned.
	out, ok := entry.GetOutputsWithMaxAge(1 * time.Hour)
	assert.True(t, ok)
	assert.NotNil(t, out)

	// With a zero maxAge, outputs should be expired.
	out, ok = entry.GetOutputsWithMaxAge(0)
	assert.False(t, ok)
	assert.Nil(t, out)
}

func TestSQLDBCacheUpsertRefreshesCreatedAt(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sp := setupTestPostgres(ctx, t)

	c := newTestSQLDBCache(t, sp)

	outputs := &wfv1.Outputs{
		Parameters: []wfv1.Parameter{
			{Name: "result", Value: wfv1.AnyStringPtr("v1")},
		},
	}
	require.NoError(t, c.Save(ctx, "upsert-key", "node-1", outputs, "1h"))

	entry1, err := c.Load(ctx, "upsert-key")
	require.NoError(t, err)
	require.NotNil(t, entry1)
	createdAt1 := entry1.CreationTimestamp.Time

	// Small delay to ensure timestamps differ.
	time.Sleep(10 * time.Millisecond)

	// Re-save with updated outputs.
	outputs2 := &wfv1.Outputs{
		Parameters: []wfv1.Parameter{
			{Name: "result", Value: wfv1.AnyStringPtr("v2")},
		},
	}
	require.NoError(t, c.Save(ctx, "upsert-key", "node-2", outputs2, "1h"))

	entry2, err := c.Load(ctx, "upsert-key")
	require.NoError(t, err)
	require.NotNil(t, entry2)

	assert.True(t, entry2.CreationTimestamp.After(createdAt1),
		"created_at should be refreshed on upsert: first=%v, second=%v", createdAt1, entry2.CreationTimestamp.Time)
	assert.Equal(t, "node-2", entry2.NodeID)
}
