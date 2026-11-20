//go:build !windows

package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	sqldbtest "github.com/argoproj/argo-workflows/v4/test/util/sqldb"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const (
	testNamespace = "default"
	testCacheName = "my-cache"
)

var testTableName = memodb.TableName(nil)

func setupTestPostgres(ctx context.Context, t *testing.T) *sqldb.SessionProxy {
	t.Helper()
	dbCfg := sqldbtest.Postgres(ctx, t)
	sp, err := sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{
		DBConfig:   dbCfg,
		Username:   sqldbtest.Username,
		Password:   sqldbtest.Password,
		MaxRetries: 5,
		BaseDelay:  200 * time.Millisecond,
		MaxDelay:   10 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sp.Close() })

	require.NoError(t, memodb.Migrate(ctx, sp, memodb.ConfigFromConfig(nil)))
	return sp
}

func newTestSQLDBCache(t *testing.T, sp *sqldb.SessionProxy) MemoizationCache {
	t.Helper()
	queries, err := memodb.NewQueries(testTableName, sp)
	require.NoError(t, err)
	return newSQLDBCache(testNamespace, testCacheName, func(fn func(memodb.MemoizationDB) error) error { return fn(queries) })
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

	require.NoError(t, c.Save(ctx, "no-outputs", "node-empty", nil, ""))
	entry, err = c.Load(ctx, "no-outputs")
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Nil(t, entry.Outputs)
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

	_, err = sp.Session(ctx).SQL().Update(testTableName).Set("outputs", "{invalid JSON").Where("cache_key = ?", "complex-key").Exec()
	require.NoError(t, err)
	entry, err = c.Load(ctx, "complex-key")
	require.ErrorContains(t, err, "malformed memoization db entry")
	assert.Nil(t, entry, "corrupt cache data must not be returned as a hit")
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
