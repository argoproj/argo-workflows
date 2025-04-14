package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_UnmarshalJSON(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		p := Plugin{}
		require.EqualError(t, p.UnmarshalJSON([]byte(`1`)), "json: cannot unmarshal number into Go value of type map[string]interface {}")
	})
	t.Run("NoKeys", func(t *testing.T) {
		p := Plugin{}
		require.EqualError(t, p.UnmarshalJSON([]byte(`{}`)), "expected exactly one key, got 0")
	})
	t.Run("OneKey", func(t *testing.T) {
		p := Plugin{}
		require.NoError(t, p.UnmarshalJSON([]byte(`{"foo":1}`)))
	})
}

func TestPlugin_Names(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		p := Plugin{}
		name, err := p.Name()
		require.EqualError(t, err, "plugin value is empty")
		assert.Empty(t, name)
	})
	t.Run("Invalid", func(t *testing.T) {
		p := Plugin{
			Object: Object{Value: json.RawMessage(`1`)},
		}
		name, err := p.Name()
		require.EqualError(t, err, "json: cannot unmarshal number into Go value of type map[string]interface {}")
		assert.Empty(t, name)
	})
	t.Run("NoKeys", func(t *testing.T) {
		p := Plugin{
			Object: Object{Value: json.RawMessage(`{}`)},
		}
		name, err := p.Name()
		require.EqualError(t, err, "expected exactly one key, got 0")
		assert.Empty(t, name)
	})
	t.Run("OneKey", func(t *testing.T) {
		p := Plugin{
			Object: Object{Value: json.RawMessage(`{"foo":1}`)},
		}
		name, err := p.Name()
		require.NoError(t, err)
		assert.Equal(t, "foo", name)
	})
}
