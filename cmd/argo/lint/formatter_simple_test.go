package lint

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleFormatter(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		msg := formatterSimple{}.Format(&LintResults{
			Success: true,
		})
		expected := "no linting errors found!\n"
		assert.Equal(t, expected, msg)
	})
	t.Run("Nothing linted", func(t *testing.T) {
		msg := formatterSimple{}.Format(&LintResults{
			anythingLinted: false,
			Success:        false,
		})
		expected := "found nothing to lint in the specified paths, failing...\n"
		assert.Equal(t, expected, msg)
	})
	t.Run("Linting Errors", func(t *testing.T) {
		msg := formatterSimple{}.Format(&LintResults{
			Results: []*LintResult{
				{
					File: "test1",
					Errs: []error{
						fmt.Errorf("some error"),
						fmt.Errorf("some error2"),
					},
					Linted: true,
				},
				{
					File: "test2",
					Errs: []error{
						fmt.Errorf("some error"),
					},
					Linted: true,
				},
				{
					File:   "test3",
					Linted: false,
				},
			},
			anythingLinted: true,
		})
		expected := `test1: some error
test1: some error2
test2: some error
`
		assert.Equal(t, expected, msg)
	})
}
