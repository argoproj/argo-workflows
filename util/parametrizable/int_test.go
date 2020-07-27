package parametrizable

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"testing"
)

var template = `
val: %s`

type testStruct struct {
	Val Int `json:"val"`
}

func unmarshal(val string) *testStruct {
	var res testStruct
	err := yaml.Unmarshal([]byte(fmt.Sprintf(template, val)), &res)
	if err != nil {
		panic(err)
	}
	return &res
}

func TestInt_UnmarshalJSON(t *testing.T) {
	val, err := unmarshal("2").Val.Int()
	assert.NoError(t, err)
	assert.Equal(t, 2, val)


	val, err = unmarshal("\"2\"").Val.Int()
	assert.NoError(t, err)
	assert.Equal(t, 2, val)

	var res *testStruct
	assert.NotPanics(t, func() {
		res = unmarshal("\"{{var}}\"")
	})
	_, err = res.Val.Int()
	assert.Error(t, err)
}
