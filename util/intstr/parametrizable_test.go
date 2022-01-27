package intstr

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInt(t *testing.T) {
	i, err := Int(ParsePtr("2"))
	fmt.Println("HERE HERE")
	val := ParsePtr("2")
	json, err := json.Marshal(val)
	fmt.Println(val)
	fmt.Println(json)
	assert.Error(t, err)
	assert.NoError(t, err)
	assert.Equal(t, 2, *i)

	i, err = Int(ParsePtr("-1"))
	assert.NoError(t, err)
	assert.Equal(t, -1, *i)

	_, err = Int(ParsePtr("{{argo.variable}}"))
	assert.Error(t, err)

	i, err = Int(nil)
	assert.NoError(t, err)
	assert.Nil(t, i)
}

func TestInt32(t *testing.T) {
	i, err := Int32(ParsePtr("2"))
	assert.NoError(t, err)
	assert.Equal(t, int32(2), *i)

	i, err = Int32(ParsePtr("-1"))
	assert.NoError(t, err)
	assert.Equal(t, int32(-1), *i)

	_, err = Int32(ParsePtr("{{argo.variable}}"))
	assert.Error(t, err)

	i, err = Int32(nil)
	assert.NoError(t, err)
	assert.Nil(t, i)
}

func TestInt64(t *testing.T) {
	i, err := Int64(ParsePtr("2"))
	assert.NoError(t, err)
	assert.Equal(t, int64(2), *i)

	i, err = Int64(ParsePtr("-1"))
	assert.NoError(t, err)
	assert.Equal(t, int64(-1), *i)

	_, err = Int64(ParsePtr("{{argo.variable}}"))
	assert.Error(t, err)

	i, err = Int64(nil)
	assert.NoError(t, err)
	assert.Nil(t, i)
}

func TestIsValidIntOrArgoVariable(t *testing.T) {
	assert.True(t, IsValidIntOrArgoVariable(ParsePtr("2")))
	assert.True(t, IsValidIntOrArgoVariable(ParsePtr("-1")))
	assert.True(t, IsValidIntOrArgoVariable(ParsePtr("{{argo.variable}}")))
	assert.True(t, IsValidIntOrArgoVariable(nil))

	assert.False(t, IsValidIntOrArgoVariable(ParsePtr("some-string")))
	assert.False(t, IsValidIntOrArgoVariable(ParsePtr("1.5")))
}
