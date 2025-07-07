package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryJSONUnmarshal(t *testing.T) {
	for _, testcase := range []struct {
		input    []string
		success  bool
		expected []interface{}
	}{
		{[]string{"1"}, false, nil},
		{[]string{"1", "2"}, false, nil},
		{[]string{"foo"}, false, nil},
		{[]string{"foo", "bar"}, false, nil},
		{[]string{`["1"]`, "2"}, false, nil},       // Fails on second element
		{[]string{`{"foo":"1"}`, "2"}, false, nil}, // Fails on second element
		{[]string{`["1"]`, `["2"]`}, true, []interface{}{[]interface{}{"1"}, []interface{}{"2"}}},
		{[]string{`["1"]`, `["2"]`}, true, []interface{}{[]interface{}{"1"}, []interface{}{"2"}}},
		{[]string{"\n[\"1\"]  \n", "\t[\"2\"]\t"}, true, []interface{}{[]interface{}{"1"}, []interface{}{"2"}}},
		{[]string{`{"number":"1"}`, `{"number":"2"}`}, true, []interface{}{map[string]interface{}{"number": "1"}, map[string]interface{}{"number": "2"}}},
		{[]string{`[{"foo":"apple", "bar":"pear"}]`, `{"foo":"banana"}`}, true, []interface{}{[]interface{}{map[string]interface{}{"bar": "pear", "foo": "apple"}}, map[string]interface{}{"foo": "banana"}}},
	} {
		t.Run(fmt.Sprintf("Unmarshal %v", testcase.input),
			func(t *testing.T) {
				list, success := tryJSONUnmarshal(testcase.input)
				require.Equal(t, testcase.success, success)
				if success {
					assert.Equal(t, testcase.expected, list)
				}
			})
	}
}

func TestAggregatedJsonValueList(t *testing.T) {
	for _, testcase := range []struct {
		input    []string
		expected string
	}{
		{[]string{"1"}, `["1"]`},
		{[]string{"1", "2"}, `["1","2"]`},
		{[]string{"foo"}, `["foo"]`},
		{[]string{"foo", "bar"}, `["foo","bar"]`},
		{[]string{`["1"]`, "2"}, `["[\"1\"]","2"]`},               // This is expected, but not really useful
		{[]string{`{"foo":"1"}`, "2"}, `["{\"foo\":\"1\"}","2"]`}, // This is expected, but not really useful
		{[]string{`["1"]`, `["2"]`}, `[["1"],["2"]]`},
		{[]string{` ["1"]`, `["2"] `}, `[["1"],["2"]]`},
		{[]string{"\n[\"1\"]  \n", "\t[\"2\"]\t"}, `[["1"],["2"]]`},
		{[]string{`{"number":"1"}`, `{"number":"2"}`}, `[{"number":"1"},{"number":"2"}]`},
		{[]string{`[{"foo":"apple", "bar":"pear"}]`}, `[[{"bar":"pear","foo":"apple"}]]`}, // Sorted map keys here may make this a fragile test, can be dropped
	} {
		t.Run(fmt.Sprintf("Aggregate %v", testcase.input),
			func(t *testing.T) {
				result, err := aggregatedJSONValueList(testcase.input)
				require.NoError(t, err)
				assert.Equal(t, testcase.expected, result)
			})
	}
}
