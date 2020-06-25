package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItem(t *testing.T) {
	for data, expectedType := range map[string]Type{
		"0":                               Number,
		"3.141":                           Number,
		"true":                            Bool,
		"\"hello\"":                       String,
		"{\"val\":\"123\"}":               Map,
		"[\"1\",\"2\",\"3\",\"4\",\"5\"]": List,
	} {
		t.Run(string(expectedType), func(t *testing.T) {
			t.Run("Item", func(t *testing.T) {
				runItemTest(t, data, expectedType)
			})
		})
	}
}

func runItemTest(t *testing.T, data string, expectedType Type) {
	itm, err := ParseItem(data)
	assert.NoError(t, err)
	assert.Equal(t, itm.GetType(), expectedType)
	jsonBytes, err := json.Marshal(itm)
	assert.NoError(t, err)
	assert.Equal(t, data, string(jsonBytes), "marshalling is symmetric")
	if strings.HasPrefix(data, `"`) {
		assert.Equal(t, data, fmt.Sprintf("\"%v\"", itm))
		assert.Equal(t, data, fmt.Sprintf("\"%s\"", itm))
	} else {
		assert.Equal(t, data, fmt.Sprintf("%v", itm))
		assert.Equal(t, data, fmt.Sprintf("%s", itm))
	}
}

func TestItem_GetMapVal(t *testing.T) {
	item := Item{}
	err := json.Unmarshal([]byte(`{"foo":"bar"}`), &item)
	assert.NoError(t, err)
	val := item.GetMapVal()
	assert.Len(t, val, 1)
}

func TestItem_GetListVal(t *testing.T) {
	item := Item{}
	err := json.Unmarshal([]byte(`["foo"]`), &item)
	assert.NoError(t, err)
	val := item.GetListVal()
	assert.Len(t, val, 1)
}

func TestItem_GetStrVal(t *testing.T) {
	item := Item{}
	err := json.Unmarshal([]byte(`"foo"`), &item)
	assert.NoError(t, err)
	val := item.GetStrVal()
	assert.Equal(t, "foo", val)
}