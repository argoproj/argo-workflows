package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnumFlagValue(t *testing.T) {
	e := EnumFlagValue{
		AllowedValues: []string{"name", "json", "yaml", "wide"},
		Value:         "json",
	}
	t.Run("Usage", func(t *testing.T) {
		assert.Equal(t, "One of: name|json|yaml|wide", e.Usage())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "json", e.String())
	})

	t.Run("Type", func(t *testing.T) {
		assert.Equal(t, "string", e.Type())
	})

	t.Run("Set", func(t *testing.T) {
		err := e.Set("name")
		require.NoError(t, err)
		assert.Equal(t, "name", e.Value)

		err = e.Set("invalid")
		require.Error(t, err, "One of: name|json|yaml|wide")
	})
}
