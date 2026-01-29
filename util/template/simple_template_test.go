package template

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func simpleReplaceHelper(ctx context.Context, w io.Writer, tag string, replaceMap map[string]interface{}, allowUnresolved bool) (int, error) {
	replacement, ok := replaceMap[strings.TrimSpace(tag)]
	if !ok {
		// Attempt to resolve nested tags, if possible
		if index := strings.LastIndex(tag, "{{"); index > 0 {
			nestedTagPrefix := tag[:index]
			nestedTag := tag[index+2:]
			if replacement, ok := replaceMap[nestedTag]; ok {
				replacement, isStr := replacement.(string)
				if isStr {
					replacement = strconv.Quote(replacement)
					replacement = replacement[1 : len(replacement)-1]
					return w.Write([]byte("{{" + nestedTagPrefix + replacement))
				}
			}
		}
		if allowUnresolved {
			// just write the same string back
			logger := logging.RequireLoggerFromContext(ctx)
			logger.WithError(errors.InternalError("unresolved")).Debug(ctx, "unresolved is allowed")
			return fmt.Fprintf(w, "{{%s}}", tag)
		}
		return 0, errors.Errorf(errors.CodeBadRequest, "failed to resolve {{%s}}", tag)
	}

	replacementStr, isStr := replacement.(string)
	if !isStr {
		return 0, errors.Errorf(errors.CodeBadRequest, "failed to resolve {{%s}} to string", tag)
	}
	// The following escapes any special characters (e.g. newlines, tabs, etc...)
	// in preparation for substitution
	replacementStr = strconv.Quote(replacementStr)
	replacementStr = replacementStr[1 : len(replacementStr)-1]
	return w.Write([]byte(replacementStr))
}

func Test_CompareSimpleReplace(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	replaceMap := map[string]interface{}{"foo": "bar"}

	tests := []struct {
		tag             string
		allowUnresolved bool
	}{
		{"foo", false},
		{"foo", true},
		{"bar", false}, // unresolved
		{"bar", true},  // unresolved
		{"nested-{{foo}}", false},
		{"nested-{{bar}}", false},
		// Artifact case which might differ
		{"steps.step.outputs.artifacts.art", false},
		{"steps.step.outputs.artifacts.art", true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s_%v", tc.tag, tc.allowUnresolved), func(t *testing.T) {
			// Helper (Old logic)
			var b1 strings.Builder
			_, err1 := simpleReplaceHelper(ctx, &b1, tc.tag, replaceMap, tc.allowUnresolved)
			res1 := b1.String()

			// New logic
			var b2 strings.Builder
			_, err2 := simpleReplace(ctx, &b2, tc.tag, replaceMap, tc.allowUnresolved)
			res2 := b2.String()

			if err1 != nil {
				if err2 == nil {
					t.Errorf("Old errored (%v) but New did not (res: %s)", err1, res2)
				}
			} else {
				if err2 != nil {
					t.Errorf("Old succeeded (res: %s) but New errored (%v)", res1, err2)
				} else {
					if res1 != res2 {
						t.Errorf("Results differ: Old=%q, New=%q", res1, res2)
					}
				}
			}
		})
	}
}

func Test_SimpleReplace(t *testing.T) {

	ctx := logging.TestContext(t.Context())

	tests := []struct {
		name            string
		tag             string
		replaceMap      map[string]interface{}
		allowUnresolved bool
		expectedWritten string
		expectedError   string
	}{
		{
			name: "BasicReplacement",
			tag:  "foo",
			replaceMap: map[string]interface{}{
				"foo": "bar",
			},
			expectedWritten: "bar",
		},
		{
			name: "BasicReplacementWithWhitespace",
			tag:  " foo ",
			replaceMap: map[string]interface{}{
				"foo": "bar",
			},
			expectedWritten: "bar",
		},
		{
			name: "NestedReplacement",
			// Simulating fasttemplate behavior: "{{outer-{{inner}}}}"
			// fasttemplate sees "{{", reads until "}}".
			// It captures "outer-{{inner".
			tag: "outer-{{inner",
			replaceMap: map[string]interface{}{
				"inner": "suffix",
			},
			// Expects to write back the resolved nested tag, re-wrapped in {{ }} by the caller?
			// No, simpleReplace writes "{{" + prefix + replacement.
			// So it writes "{{outer-suffix".
			// The caller (fasttemplate) will append the closing "}}" from the original string.
			expectedWritten: "{{outer-suffix",
		},
		        {
		            name: "NestedReplacementQuoted",
		            tag:  "msg-{{val",
		            replaceMap: map[string]interface{}{
		                "val": "hello \"world\"",
		            },
		            // replacement is quoted: "hello \"world\"" -> "\\"hello \\\"world\\\"" -> inner stripped -> "hello \\\"world\\\""
		            expectedWritten: "{{msg-hello \\\"world\\\"",
		        },		{
			name: "NestedTagNotFound",
			tag:  "outer-{{unknown",
			replaceMap: map[string]interface{}{
				"inner": "suffix",
			},
			allowUnresolved: true,
			// Should fail to resolve nested, so falls back to allowing unresolved.
			// Writes "{{tag}}" -> "{{outer-{{unknown}}"
			expectedWritten: "{{outer-{{unknown}}",
		},
		{
			name: "NestedTagNotFound_DisallowUnresolved",
			tag:  "outer-{{unknown",
			replaceMap: map[string]interface{}{
				"inner": "suffix",
			},
			allowUnresolved: false,
			expectedError:   "failed to resolve {{outer-{{unknown}}",
		},
		{
			name: "DoubleNested_ResolvesLast",
			// "{{a {{b}} {{c}}}}" -> tag "a {{b}} {{c"
			tag: "a {{b}} {{c",
			replaceMap: map[string]interface{}{
				"b": "B",
				"c": "C",
			},
			// LastIndex finds {{c.
			// Writes "{{a {{b}} C".
			expectedWritten: "{{a {{b}} C",
		},
		{
			name: "OuterTagDirectMatchTakesPrecedence",
			tag:  "outer-{{inner",
			replaceMap: map[string]interface{}{
				"outer-{{inner": "direct-hit",
				"inner":         "ignored",
			},
			expectedWritten: "direct-hit",
		},
		{
			name: "TripleNested_ResolvesInnermost",
			// Input: "{{A {{B {{C}}}}"
			// fasttemplate splits at first "}}".
			// Tag content: "A {{B {{C"
			// Logic:
			// 1. LastIndex finds "{{C".
			// 2. Resolves "C".
			// 3. Writes "{{A {{B resolved_C".
			// 4. Returns.
			// The A and B tags are not resolved because they rely on the inner content being resolved first?
			// Or simply because their "name" would be "A {{B {{C" which is invalid/incomplete.
			tag: "A {{B {{C",
			replaceMap: map[string]interface{}{
				"C": "valC",
			},
			expectedWritten: "{{A {{B valC",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var w bytes.Buffer
			_, err := simpleReplace(ctx, &w, tc.tag, tc.replaceMap, tc.allowUnresolved)

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedWritten, w.String())
			}
		})
	}
}
