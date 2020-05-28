package v1alpha1

import (
	"encoding/json"
	"fmt"
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
				runItemTest(t, data, &Item{}, expectedType)
			})
			t.Run("ItemValue", func(t *testing.T) {
				runItemTest(t, data, &ItemValue{}, expectedType)
			})
		})
	}
}

func runItemTest(t *testing.T, data string, itm Typer, expectedType Type) {
	err := json.Unmarshal([]byte(data), itm)
	assert.NoError(t, err)
	assert.Equal(t, itm.GetType(), expectedType)
	jsonBytes, err := json.Marshal(itm)
	assert.NoError(t, err)
	assert.Equal(t, data, string(jsonBytes))
	if itm.GetType() == String {
		assert.Equal(t, data, fmt.Sprintf("\"%v\"", itm))
		assert.Equal(t, data, fmt.Sprintf("\"%s\"", itm))
	} else {
		assert.Equal(t, data, fmt.Sprintf("%v", itm))
		assert.Equal(t, data, fmt.Sprintf("%s", itm))
	}
}
