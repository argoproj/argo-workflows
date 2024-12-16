package sqldb

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type simpleStruct struct {
	Val         string          `json:"val"`
	Val2        *string         `json:"val2"`
	Val3        *string         `json:"val3,omitempty"`
	InnerSimple *simpleStruct   `json:"sstruct,omitempty"`
	List        []string        `json:"list"`
	List2       *[]simpleStruct `json:"list2,omitempty"`
}

func TestStrings(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	val2 := "val2 \uC582"
	val3 := "val3 \uC583"
	inner := simpleStruct{
		Val: "inner val \uc589",
	}
	l := []simpleStruct{inner}
	s := simpleStruct{
		Val:         "hello \x00",
		Val2:        &val2,
		Val3:        &val3,
		InnerSimple: &inner,
		List2:       &l,
	}

	newMap, err := convertMap(s)
	require.NoError(err)
	assert.Contains(string(newMap["val"].(string)), "\\x00")
	assert.Contains(string(newMap["val3"].(string)), "\\uc583")

	innerMapI, ok := newMap["sstruct"]
	require.True(ok)
	innerMap, ok := innerMapI.(map[string]interface{})
	require.True(ok)
	assert.Contains(string((innerMap["val"]).(string)), "inner val \\uc589")

	_, err = json.Marshal(newMap)
	require.NoError(err)

}
