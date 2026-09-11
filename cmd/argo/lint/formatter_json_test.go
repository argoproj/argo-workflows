package lint

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonSummarize(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		msg := formatterJson{}.Summarize(&LintResults{
			Success: true,
			Results: []*LintResult{
				{
					File: "test1",
					Errs: []error{
						fmt.Errorf("some error"),
					},
					Linted: true,
				},
			},
		})
		expected := "{\"results\":[{\"file\":\"test1\",\"errors\":[\"some error\"],\"linted\":true}],\"success\":true,\"anything_linted\":false}"
		assert.Equal(t, expected, msg)
	})
	t.Run("Nothing linted", func(t *testing.T) {
		msg := formatterJson{}.Summarize(&LintResults{
			anythingLinted: false,
			Success:        false,
		})
		expected := "{\"results\":[],\"success\":false,\"anything_linted\":false}"
		assert.Equal(t, expected, msg)
	})
}

func TestJsonFormat(t *testing.T) {
	t.Run("Error Exists", func(t *testing.T) {
		msg := formatterJson{}.Format(&LintResult{
			File: "test1",
			Errs: []error{
				fmt.Errorf("some error"),
				fmt.Errorf("some error2"),
			},
			Linted: true,
		})
		expected := ""
		assert.Equal(t, expected, msg)
	})
}
