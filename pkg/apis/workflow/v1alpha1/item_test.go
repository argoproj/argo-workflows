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
		"{\"val\":123}": Map,
	}

	for data, expectedType := range testData {
		var itm Item
		err := json.Unmarshal([]byte(data), &itm)
		assert.Nil(t, err)
		assert.Equal(t, itm.Type, expectedType)
		jsonBytes, err := json.Marshal(itm)
		assert.Equal(t, string(data), string(jsonBytes))
		assert.Equal(t, string(data), fmt.Sprintf("%v", itm))
		assert.Equal(t, string(data), fmt.Sprintf("%s", itm))
	}
}

// func TestFromInt(t *testing.T) {
// 	i := FromInt(93)
// 	if i.Type != Int || i.IntVal != 93 {
// 		t.Errorf("Expected IntVal=93, got %+v", i)
// 	}
// }

// func TestFromString(t *testing.T) {
// 	i := FromString("76")
// 	if i.Type != String || i.StrVal != "76" {
// 		t.Errorf("Expected StrVal=\"76\", got %+v", i)
// 	}
// }

// func TestIntOrStringUnmarshalJSON(t *testing.T) {
// 	cases := []struct {
// 		input  string
// 		result IntOrString
// 	}{
// 		{"{\"val\": 123}", FromInt(123)},
// 		{"{\"val\": \"123\"}", FromString("123")},
// 	}

// 	for _, c := range cases {
// 		var result IntOrStringHolder
// 		if err := json.Unmarshal([]byte(c.input), &result); err != nil {
// 			t.Errorf("Failed to unmarshal input '%v': %v", c.input, err)
// 		}
// 		if result.IOrS != c.result {
// 			t.Errorf("Failed to unmarshal input '%v': expected %+v, got %+v", c.input, c.result, result)
// 		}
// 	}
// }

// func TestIntOrStringMarshalJSON(t *testing.T) {
// 	cases := []struct {
// 		input  IntOrString
// 		result string
// 	}{
// 		{FromInt(123), "{\"val\":123}"},
// 		{FromString("123"), "{\"val\":\"123\"}"},
// 	}

// 	for _, c := range cases {
// 		input := IntOrStringHolder{c.input}
// 		result, err := json.Marshal(&input)
// 		if err != nil {
// 			t.Errorf("Failed to marshal input '%v': %v", input, err)
// 		}
// 		if string(result) != c.result {
// 			t.Errorf("Failed to marshal input '%v': expected: %+v, got %q", input, c.result, string(result))
// 		}
// 	}
// }

// func TestIntOrStringMarshalJSONUnmarshalYAML(t *testing.T) {
// 	cases := []struct {
// 		input IntOrString
// 	}{
// 		{FromInt(123)},
// 		{FromString("123")},
// 	}

// 	for _, c := range cases {
// 		input := IntOrStringHolder{c.input}
// 		jsonMarshalled, err := json.Marshal(&input)
// 		if err != nil {
// 			t.Errorf("1: Failed to marshal input: '%v': %v", input, err)
// 		}

// 		var result IntOrStringHolder
// 		err = yaml.Unmarshal(jsonMarshalled, &result)
// 		if err != nil {
// 			t.Errorf("2: Failed to unmarshal '%+v': %v", string(jsonMarshalled), err)
// 		}

// 		if !reflect.DeepEqual(input, result) {
// 			t.Errorf("3: Failed to marshal input '%+v': got %+v", input, result)
// 		}
// 	}
// }

// func TestGetValueFromIntOrPercent(t *testing.T) {
// 	tests := []struct {
// 		input     IntOrString
// 		total     int
// 		roundUp   bool
// 		expectErr bool
// 		expectVal int
// 	}{
// 		{
// 			input:     FromInt(123),
// 			expectErr: false,
// 			expectVal: 123,
// 		},
// 		{
// 			input:     FromString("90%"),
// 			total:     100,
// 			roundUp:   true,
// 			expectErr: false,
// 			expectVal: 90,
// 		},
// 		{
// 			input:     FromString("90%"),
// 			total:     95,
// 			roundUp:   true,
// 			expectErr: false,
// 			expectVal: 86,
// 		},
// 		{
// 			input:     FromString("90%"),
// 			total:     95,
// 			roundUp:   false,
// 			expectErr: false,
// 			expectVal: 85,
// 		},
// 		{
// 			input:     FromString("%"),
// 			expectErr: true,
// 		},
// 		{
// 			input:     FromString("90#"),
// 			expectErr: true,
// 		},
// 		{
// 			input:     FromString("#%"),
// 			expectErr: true,
// 		},
// 	}

// 	for i, test := range tests {
// 		t.Logf("test case %d", i)
// 		value, err := GetValueFromIntOrPercent(&test.input, test.total, test.roundUp)
// 		if test.expectErr && err == nil {
// 			t.Errorf("expected error, but got none")
// 			continue
// 		}
// 		if !test.expectErr && err != nil {
// 			t.Errorf("unexpected err: %v", err)
// 			continue
// 		}
// 		if test.expectVal != value {
// 			t.Errorf("expected %v, but got %v", test.expectVal, value)
// 		}
// 	}
// }
