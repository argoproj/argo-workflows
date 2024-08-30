package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnyString(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		x := AnyStringPtr("")
		data, err := json.Marshal(x)
		require.NoError(t, err)
		assert.Equal(t, `""`, string(data), "string value has quotes")

		i := AnyStringPtr("")
		err = i.UnmarshalJSON([]byte(`""`))
		require.NoError(t, err)
		assert.Equal(t, AnyStringPtr(""), i)

		assert.Equal(t, "", i.String(), "string value does not have quotes")
	})
	t.Run("String", func(t *testing.T) {
		x := AnyStringPtr("my-string")
		data, err := json.Marshal(x)
		require.NoError(t, err)
		assert.Equal(t, `"my-string"`, string(data), "string value has quotes")

		i := AnyStringPtr("")
		err = i.UnmarshalJSON([]byte(`"my-string"`))
		require.NoError(t, err)
		assert.Equal(t, AnyStringPtr("my-string"), i)

		assert.Equal(t, "my-string", i.String(), "string value does not have quotes")
	})
	t.Run("StringNumber", func(t *testing.T) {
		x := AnyStringPtr(1)
		data, err := json.Marshal(x)
		require.NoError(t, err)
		assert.Equal(t, `"1"`, string(data), "number value has quotes")

		i := AnyStringPtr("")
		err = i.UnmarshalJSON([]byte(`"1"`))
		require.NoError(t, err)
		assert.Equal(t, AnyStringPtr("1"), i)

		assert.Equal(t, "1", i.String(), "number value does not have quotes")
	})
	t.Run("LargeNumber", func(t *testing.T) {
		x := ParseAnyString(881217801864)
		data, err := json.Marshal(x)
		require.NoError(t, err)
		assert.Equal(t, `"881217801864"`, string(data))

		i := AnyStringPtr("")
		err = i.UnmarshalJSON([]byte("881217801864"))
		require.NoError(t, err)
		assert.Equal(t, AnyStringPtr("881217801864"), i)

		assert.Equal(t, "881217801864", i.String())
	})
	t.Run("FloatNumber", func(t *testing.T) {
		x := ParseAnyString(0.2)
		data, err := json.Marshal(x)
		require.NoError(t, err)
		assert.Equal(t, `"0.2"`, string(data))

		i := AnyStringPtr("")
		err = i.UnmarshalJSON([]byte("0.2"))
		require.NoError(t, err)
		assert.Equal(t, AnyStringPtr("0.2"), i)

		assert.Equal(t, "0.2", i.String())
	})
	t.Run("Boolean", func(t *testing.T) {
		x := ParseAnyString(true)
		data, err := json.Marshal(x)
		require.NoError(t, err)
		assert.Equal(t, `"true"`, string(data))

		x = ParseAnyString(false)
		data, err = json.Marshal(x)
		require.NoError(t, err)
		assert.Equal(t, `"false"`, string(data))

		i := AnyStringPtr("")
		err = i.UnmarshalJSON([]byte("true"))
		require.NoError(t, err)
		assert.Equal(t, AnyStringPtr("true"), i)

		assert.Equal(t, "true", i.String())
	})
}
