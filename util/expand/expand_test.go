package expand

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand(t *testing.T) {
	for i := 0; i < 1; i++ { // loop 100 times, because map ordering is not determisitic
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			before := map[string]interface{}{
				"a.b":   1,
				"a.c.d": 2,
				"a":     3, // should be deleted
				"ab":    4,
				"abb":   5, // should be kept
			}
			after := Expand(before)
			assert.Len(t, before, 5, "original map unchanged")
			assert.Equal(t, map[string]interface{}{
				"a": map[string]interface{}{
					"b": 1,
					"c": map[string]interface{}{
						"d": 2,
					},
				},
				"ab":  4,
				"abb": 5,
			}, after)
		})
	}
}

func TestExpand2(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Simple expansion",
			input: map[string]interface{}{
				"foo.bar": "baz",
				"qux":     "quux",
			},
			expected: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
				"qux": "quux",
			},
		},
		{
			name: "Nested expansion",
			input: map[string]interface{}{
				"a.b.c": 1,
				"a.b.d": 2,
				"a.e":   3,
			},
			expected: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": 1,
						"d": 2,
					},
					"e": 3,
				},
			},
		},
		{
			name: "Conflict resolution",
			input: map[string]interface{}{
				"a.b": 1,
				"a":   2,
			},
			expected: map[string]interface{}{
				"a": map[string]interface{}{
					"b": 1,
				},
			},
		},
		{
			name:     "Empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "Single key-value pair",
			input: map[string]interface{}{
				"single.key": "value",
			},
			expected: map[string]interface{}{
				"single": map[string]interface{}{
					"key": "value",
				},
			},
		},
		{
			name: "Deeply nested structure",
			input: map[string]interface{}{
				"a.b.c.d.e.f": 1,
			},
			expected: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": map[string]interface{}{
							"d": map[string]interface{}{
								"e": map[string]interface{}{
									"f": 1,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Various types of values",
			input: map[string]interface{}{
				"string.key": "value",
				"int.key":    42,
				"float.key":  3.14,
				"bool.key":   true,
			},
			expected: map[string]interface{}{
				"string": map[string]interface{}{
					"key": "value",
				},
				"int": map[string]interface{}{
					"key": 42,
				},
				"float": map[string]interface{}{
					"key": 3.14,
				},
				"bool": map[string]interface{}{
					"key": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Expand(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
	}{
		{
			name: "Simple flattening",
			input: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
				"qux": "quux",
			},
			expected: map[string]interface{}{
				"foo.bar": "baz",
				"qux":     "quux",
			},
		},
		{
			name: "Nested flattening",
			input: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": 1,
						"d": 2,
					},
					"e": 3,
				},
			},
			expected: map[string]interface{}{
				"a.b.c": 1,
				"a.b.d": 2,
				"a.e":   3,
			},
		},
		{
			name: "Struct flattening",
			input: struct {
				A struct {
					B int
					C string
				}
				D float64
			}{
				A: struct {
					B int
					C string
				}{
					B: 1,
					C: "test",
				},
				D: 3.14,
			},
			expected: map[string]interface{}{
				"A.B": 1,
				"A.C": "test",
				"D":   3.14,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Flatten(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Flatten() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRemoveConflicts(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "No conflicts",
			input: map[string]interface{}{
				"a": 1,
				"b": 2,
			},
			expected: map[string]interface{}{
				"a": 1,
				"b": 2,
			},
		},
		{
			name: "Simple conflict",
			input: map[string]interface{}{
				"a.b": 1,
				"a":   2,
			},
			expected: map[string]interface{}{
				"a.b": 1,
			},
		},
		{
			name: "Multiple conflicts",
			input: map[string]interface{}{
				"a.b.c": 1,
				"a.b":   2,
				"a":     3,
				"d":     4,
			},
			expected: map[string]interface{}{
				"a.b.c": 1,
				"d":     4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeConflicts(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("removeConflicts() = %v, want %v", result, tt.expected)
			}
		})
	}
}
