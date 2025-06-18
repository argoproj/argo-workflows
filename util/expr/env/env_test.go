package env

import (
	"encoding/json"
	"testing"

	"github.com/expr-lang/expr"
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
	expectedJSON := `{"key":"value","number":123}`
	assert.JSONEq(t, expectedJSON, toJSON(data))

	// Test with a slice
	sliceData := []interface{}{1, "two", 3.0}
	expectedSliceJSON := `[1,"two",3.0]`
	assert.JSONEq(t, expectedSliceJSON, toJSON(sliceData))

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

func TestExprMigrationAlternatives(t *testing.T) {
	// Test data for migration examples
	testData := map[string]interface{}{
		"inputs": map[string]interface{}{
			"parameters": map[string]interface{}{
				"message": "hello world",
				"count":   "42",
				"ratio":   "3.14",
				"name":    "test-user",
				"empty":   "",
				"status":  "ready",
				"force":   "true",
				"active":  "false",
				"text":    "foo bar baz",
				"items":   []string{"a", "b", "c"},
				"numbers": []int{1, 2, 3, 4, 5},
				"a":       "15",
				"b":       "25",
			},
		},
		"workflow": map[string]interface{}{
			"creationTimestamp": "2023-01-01T15:30:45Z",
		},
	}

	funcMap := GetFuncMap(testData)

	tests := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		// String Functions
		{"string conversion", `string(inputs.parameters.count)`, "42"},
		{"lower case", `lower(inputs.parameters.message)`, "hello world"},
		{"upper case", `upper(inputs.parameters.message)`, "HELLO WORLD"},
		{"repeat string", `repeat(inputs.parameters.message, 2)`, "hello worldhello world"},
		{"split string", `split(inputs.parameters.text, " ")`, []string{"foo", "bar", "baz"}},
		{"join array", `join(inputs.parameters.items, "-")`, "a-b-c"},
		{"has prefix", `hasPrefix(inputs.parameters.message, "hello")`, true},
		{"has suffix", `hasSuffix(inputs.parameters.message, "world")`, true},
		{"replace text", `replace(inputs.parameters.text, "bar", "BAR")`, "foo BAR baz"},
		{"trim spaces", `trim("  hello  ")`, "hello"},
		{"trim custom chars", `trim("__hello__", "_")`, "hello"},

		// String Functions - Contains alternatives (since contains() isn't available)
		{"indexOf for contains positive", `indexOf(inputs.parameters.message, "world") >= 0`, true},
		{"indexOf for contains negative", `indexOf(inputs.parameters.message, "xyz") == -1`, true},

		// Math Functions
		{"addition arithmetic", `int(inputs.parameters.a) + int(inputs.parameters.b)`, int(40)},
		{"int conversion", `int(inputs.parameters.count)`, int(42)},
		{"float conversion", `float(inputs.parameters.ratio)`, float64(3.14)},
		{"max function", `max(int(inputs.parameters.a), int(inputs.parameters.b))`, int(25)},
		{"min function", `min(int(inputs.parameters.a), int(inputs.parameters.b))`, int(15)},
		{"abs function", `abs(-5)`, int(5)},
		{"ceil function", `ceil(3.2)`, float64(4)},
		{"floor function", `floor(3.8)`, float64(3)},
		{"round function", `round(3.6)`, float64(4)},

		// List Functions
		{"array indexing", `inputs.parameters.items[0]`, "a"},
		{"array slicing", `inputs.parameters.numbers[1:3]`, []int{2, 3}},
		{"array length", `len(inputs.parameters.items)`, int(3)},
		{"reverse array", `reverse(inputs.parameters.items)`, []interface{}{"c", "b", "a"}},

		// Logic/Conditionals Functions (using standard operators)
		{"logical and", `inputs.parameters.status == "ready" && int(inputs.parameters.count) > 0`, true},
		{"logical or", `inputs.parameters.empty == "" || inputs.parameters.force == "true"`, true},
		{"equality check", `inputs.parameters.status == "ready"`, true},
		{"inequality check", `inputs.parameters.status != "pending"`, true},
		{"less than", `int(inputs.parameters.a) < int(inputs.parameters.b)`, true},
		{"greater than", `int(inputs.parameters.b) > int(inputs.parameters.a)`, true},
		{"ternary operator", `inputs.parameters.empty == "" ? "default" : inputs.parameters.empty`, "default"},

		// Type Functions
		{"type check int", `type(42)`, "int"},
		{"type check string", `type("hello")`, "string"},

		// JSON Functions
		{"toJSON function", `toJSON({"name": "John"})`, "{\n  \"name\": \"John\"\n}"},
		{"fromJSON function", `fromJSON("{\"name\": \"John\"}")`, map[string]interface{}{"name": "John"}},

		// Base64 Functions
		{"toBase64 function", `toBase64("Hello World")`, "SGVsbG8gV29ybGQ="},
		{"fromBase64 function", `fromBase64("SGVsbG8gV29ybGQ=")`, "Hello World"},

		// Get function as safe accessor
		{"get function on array", `get(inputs.parameters.items, 1)`, "b"},
		{"get function on map", `get(inputs.parameters, "count")`, "42"},

		// Date/Time Functions (using available functions)
		{"now function", `now()`, ""},                                   // Will be current time
		{"date function basic", `date("2023-01-01")`, ""},               // Will be parsed time
		{"date function with time", `date("2023-01-01T15:30:45Z")`, ""}, // Will be parsed time
		{"duration function", `duration("1h")`, ""},                     // Will be duration object
		{"timezone function", `timezone("UTC")`, ""},                    // Will be timezone object
		{"workflow timestamp access", `workflow.creationTimestamp`, "2023-01-01T15:30:45Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := expr.Compile(tt.expr, expr.Env(funcMap))
			require.NoError(t, err, "Expression should compile: %s", tt.expr)

			result, err := expr.Run(program, testData)
			require.NoError(t, err, "Expression should execute: %s", tt.expr)

			// Skip time-based tests that return current time/objects
			if tt.name == "now function" || tt.name == "date function basic" ||
				tt.name == "date function with time" || tt.name == "duration function" ||
				tt.name == "timezone function" {
				assert.NotNil(t, result, "Time/duration function should return non-nil result")
				return
			}

			// Skip time-based tests that return current time
			if tt.name == "workflow timestamp access" {
				assert.Equal(t, tt.expected, result, "Time expression should produce expected result: %s", tt.expr)
			} else {
				assert.Equal(t, tt.expected, result, "Expression should produce expected result: %s", tt.expr)
			}
		})
	}
}

func TestDeprecatedSprigFunctionExamples(t *testing.T) {
	// Test data for deprecated function examples from the migration guide
	testData := map[string]interface{}{
		"inputs": map[string]interface{}{
			"parameters": map[string]interface{}{
				"count":   "5",
				"name":    "test-user",
				"text":    "foo bar baz",
				"a":       "10",
				"b":       "15",
				"str_num": "42",
				"items":   []string{"apple", "orange", "grape"},
				"status":  "ready",
				"value":   "",
				"force":   "true",
				"active":  "true",
				"ratio":   "3.14",
			},
		},
		"workflow": map[string]interface{}{
			"creationTimestamp": "2023-01-01T15:30:45Z",
		},
	}

	funcMap := GetFuncMap(testData)

	examples := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		// String operations from migration guide documentation
		{"string conversion", `string(inputs.parameters.count)`, "5"},
		{"string to lower", `lower(inputs.parameters.name)`, "test-user"},
		{"string replacement", `replace(inputs.parameters.text, "bar", "BAR")`, "foo BAR baz"},

		// Math operations from migration guide documentation
		{"arithmetic addition", `int(inputs.parameters.a) + int(inputs.parameters.b)`, int(25)},
		{"string to int conversion", `int(inputs.parameters.str_num)`, int(42)},

		// List operations from migration guide documentation
		{"array first element", `inputs.parameters.items[0]`, "apple"},
		{"array literal alternative", `["a", "b", "c"]`, []interface{}{"a", "b", "c"}},

		// Logic and comparison operations from migration guide documentation
		{"comparison and logic", `inputs.parameters.status == "ready" && int(inputs.parameters.count) > 0`, true},
		{"comparison or logic", `inputs.parameters.value == "" || inputs.parameters.force == "true"`, true},

		// Conditional logic from migration guide documentation
		{"default value ternary", `inputs.parameters.name != "" ? inputs.parameters.name : "unknown"`, "test-user"},
		{"ternary enabled/disabled", `inputs.parameters.active == "true" ? "enabled" : "disabled"`, "enabled"},

		// Type conversions from migration guide documentation
		{"string conversion explicit", `string(inputs.parameters.count)`, "5"},
		{"float conversion", `float(inputs.parameters.ratio)`, float64(3.14)},

		// Time operations from migration guide documentation - only test formats that work
		{"unix timestamp as string", `string(now().Unix())`, ""}, // Will be tested for type only
		{"workflow creation timestamp access", `workflow.creationTimestamp`, "2023-01-01T15:30:45Z"},
	}

	for _, tt := range examples {
		t.Run(tt.name, func(t *testing.T) {
			program, err := expr.Compile(tt.expr, expr.Env(funcMap))
			require.NoError(t, err, "Migration example should compile: %s", tt.expr)

			result, err := expr.Run(program, testData)
			require.NoError(t, err, "Migration example should execute: %s", tt.expr)

			// Special handling for time-based tests
			if tt.name == "unix timestamp as string" {
				assert.IsType(t, "", result, "Should return string for Unix timestamp")
				// Check that it's a valid number string
				assert.Regexp(t, `^\d+$`, result, "Unix timestamp should be numeric string")
			} else {
				assert.Equal(t, tt.expected, result, "Migration example should produce expected result: %s", tt.expr)
			}
		})
	}
}

func TestAllDocumentedMigrationExamples(t *testing.T) {
	// This test covers every specific migration example from the documentation
	testData := map[string]interface{}{
		"inputs": map[string]interface{}{
			"parameters": map[string]interface{}{
				"count":   "5",
				"name":    "test-user",
				"text":    "foo bar baz",
				"a":       "10",
				"b":       "15",
				"str_num": "42",
				"items":   []string{"apple", "orange", "grape"},
				"status":  "ready",
				"value":   "",
				"force":   "true",
				"active":  "true",
				"ratio":   "3.14",
			},
		},
		"workflow": map[string]interface{}{
			"creationTimestamp": "2023-01-01T15:30:45Z",
		},
	}

	funcMap := GetFuncMap(testData)

	// All the specific examples from docs/variables.md migration section
	examples := []struct {
		name      string
		expr      string
		expected  interface{}
		skipExact bool // For time-based tests where we check type/format instead
	}{
		// From "String operations" section
		{"doc example - string conversion", `string(inputs.parameters.count)`, "5", false},
		{"doc example - lower case", `lower(inputs.parameters.name)`, "test-user", false},
		{"doc example - replace text", `replace(inputs.parameters.text, "foo", "bar")`, "bar bar baz", false},

		// From "Math operations" section
		{"doc example - addition", `int(inputs.parameters.a) + int(inputs.parameters.b)`, int(25), false},
		{"doc example - int conversion", `int(inputs.parameters.str_num)`, int(42), false},

		// From "List operations" section
		{"doc example - first element", `inputs.parameters.items[0]`, "apple", false},
		{"doc example - array literal", `["a", "b", "c"]`, []interface{}{"a", "b", "c"}, false},

		// From "Logic and comparison operations" section
		{"doc example - and comparison", `inputs.parameters.status == "ready" && int(inputs.parameters.count) > 0`, true, false},
		{"doc example - or comparison", `inputs.parameters.value == "" || inputs.parameters.force == "true"`, true, false},

		// From "Conditional logic" section
		{"doc example - default value", `inputs.parameters.name != "" ? inputs.parameters.name : "unknown"`, "test-user", false},
		{"doc example - ternary enabled", `inputs.parameters.active == "true" ? "enabled" : "disabled"`, "enabled", false},

		// From "Type conversions" section
		{"doc example - string conversion explicit", `string(inputs.parameters.count)`, "5", false},
		{"doc example - float conversion", `float(inputs.parameters.ratio)`, float64(3.14), false},

		// From "Date/time operations" and "Common time formatting patterns" - working examples only
		{"doc example - now format basic", `now().Format("2006-01-02")`, "", true},
		{"doc example - now format ISO", `now().Format("2006-01-02T15:04:05Z07:00")`, "", true},
		{"doc example - now format readable", `now().Format("January 2, 2006")`, "", true},
		{"doc example - now format log", `now().Format("2006-01-02 15:04:05")`, "", true},
		{"doc example - now format file safe", `now().Format("20060102-150405")`, "", true},
		{"doc example - unix timestamp", `now().Unix()`, int64(0), true},
		{"doc example - unix timestamp string", `string(now().Unix())`, "", true},
		{"doc example - workflow timestamp", `workflow.creationTimestamp`, "2023-01-01T15:30:45Z", false},
	}

	for _, tt := range examples {
		t.Run(tt.name, func(t *testing.T) {
			program, err := expr.Compile(tt.expr, expr.Env(funcMap))
			require.NoError(t, err, "Documented example should compile: %s", tt.expr)

			result, err := expr.Run(program, testData)
			require.NoError(t, err, "Documented example should execute: %s", tt.expr)

			if tt.skipExact {
				// For time-based tests, just verify the type and reasonable format
				switch tt.name {
				case "doc example - unix timestamp":
					assert.IsType(t, int64(0), result, "Unix timestamp should be int64")
					assert.Greater(t, result.(int64), int64(1600000000), "Unix timestamp should be reasonable")
				case "doc example - unix timestamp string":
					assert.IsType(t, "", result, "Unix timestamp string should be string")
					assert.Regexp(t, `^\d+$`, result, "Unix timestamp string should be numeric")
				default:
					// Time format strings
					assert.IsType(t, "", result, "Time format should return string: %s", tt.expr)
					assert.NotEmpty(t, result, "Time format should not be empty: %s", tt.expr)
				}
			} else {
				assert.Equal(t, tt.expected, result, "Documented example should work as specified: %s", tt.expr)
			}
		})
	}
}

func TestTimeFormattingExamples(t *testing.T) {
	// Test time formatting patterns from migration guide
	testData := map[string]interface{}{
		"workflow": map[string]interface{}{
			"creationTimestamp": "2023-01-01T15:30:45Z",
		},
	}

	funcMap := GetFuncMap(testData)

	timeTests := []struct {
		name     string
		expr     string
		expected string
	}{
		// Only test with current time and basic formatting since time() function works
		{"current time ISO format", `now().Format("2006-01-02T15:04:05Z07:00")`, ""},
		{"current time readable", `now().Format("January 2, 2006")`, ""},
		{"current time log format", `now().Format("2006-01-02 15:04:05")`, ""},
		{"current time file safe", `now().Format("20060102-150405")`, ""},
		{"workflow timestamp access", `workflow.creationTimestamp`, "2023-01-01T15:30:45Z"},
	}

	for _, tt := range timeTests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := expr.Compile(tt.expr, expr.Env(funcMap))
			require.NoError(t, err, "Time expression should compile: %s", tt.expr)

			result, err := expr.Run(program, testData)
			require.NoError(t, err, "Time expression should execute: %s", tt.expr)

			// Skip time-based tests that return current time
			if tt.name == "workflow timestamp access" {
				assert.Equal(t, tt.expected, result, "Time expression should produce expected result: %s", tt.expr)
			} else {
				assert.IsType(t, "", result, "Should return string for time format")
			}
		})
	}
}
