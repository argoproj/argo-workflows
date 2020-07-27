package parametrizable

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
	"testing"
)

var template = `
val: %s`

type testStruct struct {
	Val Int `json:"val"`
}

func unmarshal(val string) (*testStruct, error) {
	var res testStruct
	err := yaml.Unmarshal([]byte(fmt.Sprintf(template, val)), &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func TestInt_UnmarshalJSON(t *testing.T) {
	val, err := unmarshal("2")
	assert.NoError(t, err)
	i, err := val.Val.Int()
	assert.NoError(t, err)
	assert.Equal(t, 2, i)

	val, err = unmarshal("\"2\"")
	assert.NoError(t, err)
	i, err = val.Val.Int()
	assert.NoError(t, err)
	assert.Equal(t, 2, i)


	val, err = unmarshal("2.1")
	assert.Error(t, err)
	assert.EqualError(t, err, "error unmarshaling JSON: 2.1 is not an int or argo variable")

	val, err = unmarshal("\"{{var}}\"")
	assert.NoError(t, err)
	i, err = val.Val.Int()
	assert.Error(t, err)
}
