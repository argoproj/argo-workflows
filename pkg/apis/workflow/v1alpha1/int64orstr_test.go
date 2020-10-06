package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInt64OrString(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		x := Int64OrStringPtr("")
		data, err := json.Marshal(x)
		if assert.NoError(t, err) {
			assert.Equal(t, `""`, string(data), "string value has quotes")
		}
		i := Int64OrStringPtr("")
		err = i.UnmarshalJSON([]byte(`""`))
		if assert.NoError(t, err) {
			assert.Equal(t, Int64OrStringPtr(""), i)
		}
		assert.Equal(t, "", i.String(), "string value does not have quotes")
	})
	t.Run("String", func(t *testing.T) {
		x := Int64OrStringPtr("my-string")
		data, err := json.Marshal(x)
		if assert.NoError(t, err) {
			assert.Equal(t, `"my-string"`, string(data), "string value has quotes")
		}
		i := Int64OrStringPtr("")
		err = i.UnmarshalJSON([]byte(`"my-string"`))
		if assert.NoError(t, err) {
			assert.Equal(t, Int64OrStringPtr("my-string"), i)
		}
		assert.Equal(t, "my-string", i.String(), "string value does not have quotes")
	})
	t.Run("StringNumber", func(t *testing.T) {
		x := Int64OrStringPtr(1)
		data, err := json.Marshal(x)
		if assert.NoError(t, err) {
			assert.Equal(t, `"1"`, string(data), "number value has quotes")
		}
		i := Int64OrStringPtr("")
		err = i.UnmarshalJSON([]byte(`"1"`))
		if assert.NoError(t, err) {
			assert.Equal(t, Int64OrStringPtr("1"), i)
		}
		assert.Equal(t, "1", i.String(), "number value does not have quotes")
	})
	t.Run("LargeNumber", func(t *testing.T) {
		x := ParseInt64OrString(881217801864)
		data, err := json.Marshal(x)
		if assert.NoError(t, err) {
			assert.Equal(t, `"881217801864"`, string(data))
		}
		i := Int64OrStringPtr("")
		err = i.UnmarshalJSON([]byte("881217801864"))
		if assert.NoError(t, err) {
			assert.Equal(t, Int64OrStringPtr("881217801864"), i)
		}
		assert.Equal(t, "881217801864", i.String())
	})
}
