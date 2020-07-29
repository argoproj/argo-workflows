package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
)

func TestArtifactRepository_IsArchiveLogs(t *testing.T) {
	assert.False(t, (&ArtifactRepository{}).IsArchiveLogs())
	assert.False(t, (&ArtifactRepository{ArchiveLogs: pointer.BoolPtr(false)}).IsArchiveLogs())
	assert.True(t, (&ArtifactRepository{ArchiveLogs: pointer.BoolPtr(true)}).IsArchiveLogs())
}

func TestDatabaseConfig_GetOptions(t *testing.T) {
	t.Run("Options", func(t *testing.T) {
		c := DatabaseConfig{}
		assert.Empty(t, c.GetOptions())
	})

	t.Run("SSLMode", func(t *testing.T) {
		c := DatabaseConfig{SSLMode: "disable"}
		assert.Equal(t, map[string]string{"sslmode": "disable"}, c.GetOptions())
	})
}

func TestDatabaseConfig_GetTableName(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		c := DatabaseConfig{}
		assert.Equal(t, "argo_workflows", c.GetTableName())
	})
	t.Run("OtherTableName", func(t *testing.T) {
		c := DatabaseConfig{TableName: "my-tablename"}
		assert.Equal(t, "my-tablename", c.GetTableName())
	})
}
