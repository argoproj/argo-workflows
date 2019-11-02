package v1alpha1

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItem(t *testing.T) {
	testData := map[string]Type{
		"0":             Number,
		"3.141":         Number,
		"true":          Bool,
		"\"hello\"":     String,
		"{\"val\":\"123\"}": Map,
		"[\"1\",\"2\",\"3\",\"4\",\"5\"]" : List,
	}

	for data, expectedType := range testData {
		var itm Item
		err := json.Unmarshal([]byte(data), &itm)
     		assert.Nil(t, err)
		assert.Equal(t, itm.Type, expectedType)
		jsonBytes, err := json.Marshal(itm)
		assert.Equal(t, string(data), string(jsonBytes))
		if itm.Type == String {
			assert.Equal(t, string(data), fmt.Sprintf("\"%v\"", itm))
			assert.Equal(t, string(data), fmt.Sprintf("\"%s\"", itm))
		}else {
			assert.Equal(t, string(data), fmt.Sprintf("%v", itm))
			assert.Equal(t, string(data), fmt.Sprintf("%s", itm))
		}
	}
}


