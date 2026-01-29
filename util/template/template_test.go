package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type SimpleValue struct {
	Value string `json:"value,omitempty"`
}

func processTemplate(t *testing.T, tmpl SimpleValue, replaceMap map[string]string) SimpleValue {
	tmplBytes, err := json.Marshal(tmpl)
	require.NoError(t, err)
	ctx := logging.TestContext(t.Context())
	r, err := Replace(ctx, string(tmplBytes), replaceMap, true)
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

	t.Run("NestedVariableResolution", func(t *testing.T) {
		// This test demonstrates that standard Replace requires multiple passes
		// to resolve nested tags where the outer tag depends on the inner tag's value.
		ctx := logging.TestContext(t.Context())
		// Input must be valid JSON.
		input := `{"key": "{{outer-{{inner}}}}"}`
		replaceMap := map[string]string{
			"inner":        "suffix",
			"outer-suffix": "final-value",
		}

		// Pass 1: Resolves inner tag
		// Input: ... "{{outer-{{inner}}}}" ...
		// Resolves "inner" -> "suffix".
		// Note: simpleReplace writes the resolved value.
		// Result: ... "{{outer-suffix}}" ...
		pass1, err := Replace(ctx, input, replaceMap, true)
		require.NoError(t, err)
		assert.JSONEq(t, `{"key": "{{outer-suffix}}"}`, pass1, "First pass should only resolve the inner tag")

		// Pass 2: Resolves outer tag
		// Input: ... "{{outer-suffix}}" ...
		// Resolves "outer-suffix" -> "final-value".
		pass2, err := Replace(ctx, pass1, replaceMap, true)
		require.NoError(t, err)
		assert.JSONEq(t, `{"key": "final-value"}`, pass2, "Second pass should resolve the now-valid outer tag")
	})

	t.Run("TripleNestedVariableResolution", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		// Input: { "key": "{{A {{B {{C}}}}}}" }
		// We need enough closing braces for all 3 tags (C, B, A) to eventually close.
		// Value: "{{A {{B {{C}}}}}}"
		// JSON: {"key": "..."}
		input := `{"key": "{{A {{B {{C}}}}}}"}`
		replaceMap := map[string]string{
			"C":      "valC",
			"B valC": "valB",  // The tag becomes "{{B valC}}" -> resolves to "valB"
			"A valB": "final", // The tag becomes "{{A valB}}" -> resolves to "final"
		}

		// Pass 1: Resolves innermost C
		// "{{A {{B {{C}}}}}}" -> "{{A {{B valC}}}}"
		pass1, err := Replace(ctx, input, replaceMap, true)
		require.NoError(t, err)
		assert.JSONEq(t, `{"key": "{{A {{B valC}}}}"}`, pass1)

		// Pass 2: Resolves middle B
		// "{{A {{B valC}}}}" -> "{{A valB}}"
		pass2, err := Replace(ctx, pass1, replaceMap, true)
		require.NoError(t, err)
		assert.JSONEq(t, `{"key": "{{A valB}}"}`, pass2)

		// Pass 3: Resolves outer A
		// "{{A valB}}" -> "final"
		pass3, err := Replace(ctx, pass2, replaceMap, true)
		require.NoError(t, err)
		assert.JSONEq(t, `{"key": "final"}`, pass3)
	})

}
