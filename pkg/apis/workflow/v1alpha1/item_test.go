package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItem(t *testing.T) {
	for data, expectedType := range map[string]Type{
		"0":                               Number,
		"3.141":                           Number,
		"true":                            Bool,
		"\"hello\"":                       String,
		"\"hell%test%o\"":                 String,
		"{\"val\":\"123\"}":               Map,
		"[\"1\",\"2\",\"3\",\"4\",\"5\"]": List,
	} {
		t.Run(fmt.Sprintf("%v", expectedType), func(t *testing.T) {
			t.Run("Item", func(t *testing.T) {
				runItemTest(t, data, expectedType)
			})
		})
	}
}

func runItemTest(t *testing.T, data string, expectedType Type) {
	itm, err := ParseItem(data)
	require.NoError(t, err)
	assert.Equal(t, expectedType, itm.GetType())
	jsonBytes, err := json.Marshal(itm)
	require.NoError(t, err)
	assert.JSONEq(t, data, string(jsonBytes), "marshalling is symmetric")
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
	MustUnmarshal([]byte(`{"foo":"bar"}`), &item)
	val := item.GetMapVal()
	assert.Equal(t, map[string]Item{"foo": {Value: []byte(`"bar"`)}}, val)
}

func TestItem_GetListVal(t *testing.T) {
	item := Item{}
	MustUnmarshal([]byte(`["foo"]`), &item)
	val := item.GetListVal()
	assert.Equal(t, []Item{{Value: []byte(`"foo"`)}}, val)
}

func TestItem_GetStrVal(t *testing.T) {
	item := Item{}
	MustUnmarshal([]byte(`"foo"`), &item)
	val := item.GetStrVal()
	assert.Equal(t, "foo", val)
}

var testItemStringTable = []struct {
	name   string
	origin any
	str    string
}{
	{"json-string", []string{`{"foo": "bar"}`}, `["{\"foo\": \"bar\"}"]`},
	{"flaw-string", "<&>", `<&>`},
	{"array", []int{1, 2, 3}, "[1,2,3]"},
	{"flaw-array", []string{"<&>"}, `["<&>"]`},
	{"flaw-map", map[string]string{"foo": "<&>"}, `{"foo":"<&>"}`},
	{"number", 1.1, "1.1"},
}

func TestItem_String(t *testing.T) {
	for _, s := range testItemStringTable {
		t.Run(s.name, func(t *testing.T) {
			bytes, _ := json.Marshal(s.origin)
			var i Item
			MustUnmarshal(bytes, &i)
			assert.Equal(t, s.str, i.String())
		})
	}
}
