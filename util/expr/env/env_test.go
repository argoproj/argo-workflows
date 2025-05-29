package env

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFuncMap(t *testing.T) {
	inputMap := map[string]interface{}{
		"workflow": map[string]interface{}{
			"name": "test-workflow",
		},
		"value": 123,
	}

	funcMap := GetFuncMap(inputMap)

	// Check core functions are present
	assert.NotNil(t, funcMap["asInt"], "asInt should be present")
	assert.NotNil(t, funcMap["asFloat"], "asFloat should be present")
	assert.NotNil(t, funcMap["jsonpath"], "jsonpath should be present")
	assert.NotNil(t, funcMap["toJson"], "toJson should be present")

	// Check sprig map is present
	assert.NotNil(t, funcMap["sprig"], "sprig should be present")
	sprigSubMap, ok := funcMap["sprig"].(map[string]interface{})
	require.True(t, ok, "sprig should be a map[string]interface{}")
	assert.NotEmpty(t, sprigSubMap, "sprig map should not be empty")

	// Check input map values are expanded and present
	// Access nested map directly, expand.Expand doesn't flatten to dot notation here.
	wfMap, ok := funcMap["workflow"].(map[string]interface{})
	require.True(t, ok, "workflow map should be present")
	assert.Equal(t, "test-workflow", wfMap["name"], "Workflow name should be present in nested map")
	assert.Equal(t, 123, funcMap["value"], "Input value should be present")
}

func TestSprigFuncMapAllowlist(t *testing.T) {
	// Get the sprig map via GetFuncMap
	funcMap := GetFuncMap(map[string]interface{}{})
	sprigSubMap, ok := funcMap["sprig"].(map[string]interface{})
	require.True(t, ok, "sprig should be a map[string]interface{}")

	// Check some allowed functions are present
	assert.Contains(t, sprigSubMap, "randAlphaNum", "Allowed sprig function randAlphaNum should be present")
	assert.Contains(t, sprigSubMap, "uuidv4", "Allowed sprig function uuidv4 should be present")
	assert.Contains(t, sprigSubMap, "regexReplaceAll", "Allowed sprig function regexReplaceAll should be present")
	assert.Contains(t, sprigSubMap, "merge", "Allowed sprig function merge should be present")
	assert.Contains(t, sprigSubMap, "semverCompare", "Allowed sprig function semverCompare should be present")

	// Check explicitly disallowed functions are NOT present
	assert.NotContains(t, sprigSubMap, "env", "Disallowed sprig function env should NOT be present")
	assert.NotContains(t, sprigSubMap, "expandenv", "Disallowed sprig function expandenv should NOT be present")
	assert.NotContains(t, sprigSubMap, "getHostByName", "Disallowed sprig function getHostByName should NOT be present")

	// Check functions removed because they exist in expr builtins are NOT present
	assert.NotContains(t, sprigSubMap, "add", "Sprig function 'add' should NOT be present (use expr '+')")
	assert.NotContains(t, sprigSubMap, "lower", "Sprig function 'lower' should NOT be present (use expr 'lower()')")
	assert.NotContains(t, sprigSubMap, "repeat", "Sprig function 'repeat' should NOT be present (use expr 'repeat()')")
	assert.NotContains(t, sprigSubMap, "split", "Sprig function 'split' should NOT be present (use expr 'split()')")
	assert.NotContains(t, sprigSubMap, "toString", "Sprig function 'toString' should NOT be present (use expr 'string()')")
}

func TestToJson(t *testing.T) {
	// Test with a simple map
	data := map[string]interface{}{"key": "value", "number": 123}
	expectedJson := `{"key":"value","number":123}`
	assert.JSONEq(t, expectedJson, toJSON(data))

	// Test with a slice
	sliceData := []interface{}{1, "two", 3.0}
	expectedSliceJson := `[1,"two",3.0]`
	assert.JSONEq(t, expectedSliceJson, toJSON(sliceData))

	// Test with a simple string (should be JSON-encoded string)
	assert.Equal(t, `"hello"`, toJSON("hello"))
}

func TestJsonPath(t *testing.T) {
	jsonStr := mustJSON(t, map[string]interface{}{
		"store": map[string]interface{}{
			"book": []interface{}{
				map[string]interface{}{
					"category": "reference",
					"author":   "Nigel Rees",
					"title":    "Sayings of the Century",
					"price":    8.95,
				},
				map[string]interface{}{
					"category": "fiction",
					"author":   "Evelyn Waugh",
					"title":    "Sword of Honour",
					"price":    12.99,
				},
			},
			"bicycle": map[string]interface{}{
				"color": "red",
				"price": 19.95,
			},
		},
		"expensive": 10,
	})

	// Test extracting a single value
	result := jsonPath(jsonStr, "$.store.bicycle.color")
	assert.Equal(t, "red", result)

	// Test extracting a number
	result = jsonPath(jsonStr, "$.store.book[1].price")
	assert.InEpsilon(t, 12.99, result, 0.0001)

	// Test extracting an array element
	result = jsonPath(jsonStr, "$.store.book[0]")
	expectedBook := map[string]interface{}{
		"category": "reference",
		"author":   "Nigel Rees",
		"title":    "Sayings of the Century",
		"price":    8.95,
	}
	assert.Equal(t, expectedBook, result)

	// Test with an invalid path (should panic)
	assert.Panics(t, func() {
		jsonPath(jsonStr, "$.invalid.path")
	}, "Accessing invalid path should panic")

	// Test with invalid JSON (should panic)
	assert.Panics(t, func() {
		jsonPath(`{"key": invalid}`, "$.key")
	}, "Using invalid JSON should panic")
}

func mustJSON(t *testing.T, value map[string]any) string {
	t.Helper()
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
