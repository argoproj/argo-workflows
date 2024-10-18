package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SimpleValue struct {
	Value string `json:"value,omitempty"`
}

func processTemplate(t *testing.T, tmpl SimpleValue, replaceMap map[string]string) SimpleValue {
	tmplBytes, err := json.Marshal(tmpl)
	require.NoError(t, err)
	r, err := Replace(string(tmplBytes), replaceMap, true)
	require.NoError(t, err)
	var newTmpl SimpleValue
	err = json.Unmarshal([]byte(r), &newTmpl)
	require.NoError(t, err)
	return newTmpl
}

func Test_Template_Replace(t *testing.T) {
	t.Run("ExpressionWithEscapedCharacters", func(t *testing.T) {
		testCases := map[string]struct {
			input, want string
		}{
			"ExprSingleQuotes":             {input: "{{='test'}}", want: "test"},
			"ExprDoubleQuotes":             {input: `{{="test"}}`, want: "test"},
			"ExprEscapedBackslashInString": {input: `{{='some\\path\\with\\backslashes'}}`, want: `some\path\with\backslashes`},
			"ExprEscapedNewlineInString":   {input: `{{='some\nstring\nwith\nescaped\nnewlines'}}`, want: "some\nstring\nwith\nescaped\nnewlines"},
			"ExprNewline":                  {input: "{{=1 + \n1}}", want: "2"},
			"ExprStringAsJson":             {input: "{{=toJson('test')}}", want: `"test"`},
			"ExprObjectAsJson":             {input: "{{=toJson({test: 1})}}", want: `{"test":1}`},
			"ExprArrayAsJson":              {input: "{{=toJson([1, '2', {an: 'object'}])}}", want: `[1,"2",{"an":"object"}]`},
			"ExprSingleQuoteAsString":      {input: `{{="'"}}`, want: `'`},
			"ExprDoubleQuoteAsString":      {input: `{{='"'}}`, want: `"`},
			"ExprBoolean":                  {input: `{{=true == false}}`, want: "false"},
		}

		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				tmpl := SimpleValue{Value: tc.input}
				newTmpl := processTemplate(t, tmpl, map[string]string{})
				assert.Equal(t, tc.want, newTmpl.Value)
			})
		}
	})

	t.Run("SimpleWithEscapedCharacters", func(t *testing.T) {
		testCases := map[string]struct {
			input, want string
			replaceMap  map[string]string
		}{
			"SimpleSingleQuoteAsString": {input: `{{customParam}}`, want: `This is ' John`, replaceMap: map[string]string{"customParam": `This is ' John`}},
			"SimpleDoubleQuoteAsString": {input: `{{customParam}}`, want: `This is " John`, replaceMap: map[string]string{"customParam": `This is " John`}},
		}
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				tmpl := SimpleValue{Value: tc.input}
				newTmpl := processTemplate(t, tmpl, tc.replaceMap)
				assert.Equal(t, tc.want, newTmpl.Value)
			})
		}
	})

	t.Run("ExpressionWithJsonPath", func(t *testing.T) {
		testCases := map[string]struct {
			input, want string
		}{
			"ExprNumArrayOutput":      {input: `{{=jsonpath('{"employees": [{"name": "Baris", "age":43, "friends": ["Mo", "Jai"]}, {"name": "Mo", "age": 42, "friends": ["Baris", "Jai"]}, {"name": "Jai", "age" :44, "friends": ["Baris", "Mo"]}]}', '$.employees[*].name')}}`, want: "[\"Baris\",\"Mo\",\"Jai\"]"},
			"ExprStringArrayOutput":   {input: `{{=jsonpath('{"employees": [{"name": "Baris", "age":43, "friends": ["Mo", "Jai"]}, {"name": "Mo", "age": 42, "friends": ["Baris", "Jai"]}, {"name": "Jai", "age" :44, "friends": ["Baris", "Mo"]}]}', '$.employees[0].friends')}}`, want: "[\"Mo\",\"Jai\"]"},
			"ExprSimpleObjectOutput":  {input: `{{=jsonpath('{"employees": [{"name": "Baris", "age":43},{"name": "Mo", "age": 42}, {"name": "Jai", "age" :44}]}', '$.employees[0]')}}`, want: "{\"age\":43,\"name\":\"Baris\"}"},
			"ExprObjectArrayOutput":   {input: `{{=jsonpath('{"employees": [{"name": "Baris", "age":43},{"name": "Mo", "age": 42}, {"name": "Jai", "age" :44}]}', '$')}}`, want: "{\"employees\":[{\"age\":43,\"name\":\"Baris\"},{\"age\":42,\"name\":\"Mo\"},{\"age\":44,\"name\":\"Jai\"}]}"},
			"ExprArrayInObjectOutput": {input: `{{=jsonpath('{"employees": [{"name": "Baris", "age":43, "friends": ["Mo", "Jai"]}, {"name": "Mo", "age": 42, "friends": ["Baris", "Jai"]}, {"name": "Jai", "age" :44, "friends": ["Baris", "Mo"]}]}', '$.employees[0]')}}`, want: "{\"age\":43,\"friends\":[\"Mo\",\"Jai\"],\"name\":\"Baris\"}"},
		}

		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				tmpl := SimpleValue{Value: tc.input}
				newTmpl := processTemplate(t, tmpl, map[string]string{})
				assert.Equal(t, tc.want, newTmpl.Value)
			})
		}
	})

}
