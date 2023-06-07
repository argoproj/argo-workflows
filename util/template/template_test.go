package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type SimpleValue struct {
	Value string `json:"value,omitempty"`
}

func processTemplate(t *testing.T, tmpl SimpleValue, replaceMap map[string]string) SimpleValue {
	tmplBytes, err := json.Marshal(tmpl)
	assert.NoError(t, err)
	r, err := Replace(string(tmplBytes), replaceMap, true)
	assert.NoError(t, err)
	var newTmpl SimpleValue
	err = json.Unmarshal([]byte(r), &newTmpl)
	assert.NoError(t, err)
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
}
