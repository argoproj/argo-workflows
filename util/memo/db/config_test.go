package db_test

import (
	"testing"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

func TestTableNameDefaultsAndOverrides(t *testing.T) {
	assert.Equal(t, "cache_entries", memodb.TableName(nil))
	assert.Equal(t, "cache_entries", memodb.TableName(&config.MemoizationConfig{}))
	assert.Equal(t, "custom_cache_entries", memodb.TableName(&config.MemoizationConfig{TableName: "custom_cache_entries"}))
}

func TestConfigFromConfig(t *testing.T) {
	assert.Equal(t, memodb.Config{TableName: "cache_entries"}, memodb.ConfigFromConfig(nil))
	assert.Equal(t, memodb.Config{TableName: "custom_cache", SkipMigration: true}, memodb.ConfigFromConfig(&config.MemoizationConfig{TableName: "custom_cache", SkipMigration: true}))
}

func TestSessionProxyFromConfigErrors(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	client := fake.NewClientset()
	sp, err := memodb.SessionProxyFromConfig(ctx, client, "argo", nil)
	require.ErrorContains(t, err, "memoization configuration is required")
	assert.Nil(t, sp)

	cfg := &config.MemoizationConfig{DBConfig: config.DBConfig{PostgreSQL: &config.PostgreSQLConfig{DatabaseConfig: config.DatabaseConfig{
		UsernameSecret: apiv1.SecretKeySelector{LocalObjectReference: apiv1.LocalObjectReference{Name: "missing"}, Key: "username"},
	}}}}
	sp, err = memodb.SessionProxyFromConfig(ctx, client, "argo", cfg)
	require.ErrorContains(t, err, `secrets "missing" not found`)
	assert.Nil(t, sp)
}

func TestMigrateWithoutDatabase(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	require.NoError(t, memodb.Migrate(ctx, nil, memodb.Config{}))
	sp := &sqldb.SessionProxy{}
	require.ErrorContains(t, memodb.Migrate(ctx, sp, memodb.Config{TableName: "invalid-name", SkipMigration: true}), "invalid table name")
	require.NoError(t, memodb.Migrate(ctx, sp, memodb.Config{TableName: "valid_name", SkipMigration: true}))
}

func TestNullMemoizationDB(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	q := memodb.NullMemoizationDB
	assert.False(t, q.IsEnabled())
	record, err := q.Load(ctx, "ns", "cache", "key")
	require.NoError(t, err)
	assert.Nil(t, record)
	require.NoError(t, q.Save(ctx, "ns", "cache", "key", "node", nil, ""))
	pruned, err := q.Prune(ctx)
	require.NoError(t, err)
	assert.Zero(t, pruned)
}

func TestNewQueriesRejectsInvalidTableName(t *testing.T) {
	queries, err := memodb.NewQueries("invalid-table-name", nil)
	require.Error(t, err)
	assert.Nil(t, queries)
}
