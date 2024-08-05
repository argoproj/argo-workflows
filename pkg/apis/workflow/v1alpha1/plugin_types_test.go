package v1alpha1

import (
	"testing"

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
