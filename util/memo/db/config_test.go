package db_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/config"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

func TestTableNameDefaultsAndOverrides(t *testing.T) {
	assert.Equal(t, "cache_entries", memodb.TableName(nil))
	assert.Equal(t, "cache_entries", memodb.TableName(&config.MemoizationConfig{}))
	assert.Equal(t, "custom_cache_entries", memodb.TableName(&config.MemoizationConfig{TableName: "custom_cache_entries"}))
}

func TestNewQueriesRejectsInvalidTableName(t *testing.T) {
	queries, err := memodb.NewQueries("invalid-table-name", nil)
	require.Error(t, err)
	assert.Nil(t, queries)
}
