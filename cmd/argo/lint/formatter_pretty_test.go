package lint

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrettyFormatter(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		msg := formatterPretty{}.Format(&LintResults{
			Success: true,
		})
		expected := fmt.Sprintf("%s no linting errors found!\n", withAttribute("✔", fgGreen))
		assert.Equal(t, expected, msg)
	})
	t.Run("Nothing linted", func(t *testing.T) {
		msg := formatterPretty{}.Format(&LintResults{
			anythingLinted: false,
			Success:        false,
		})
		expected := fmt.Sprintf("%s\n", withAttribute("✖ found nothing to lint in the specified paths, failing...", fgRed))
		assert.Equal(t, expected, msg)
	})
	t.Run("Linting Errors", func(t *testing.T) {
		msg := formatterPretty{}.Format(&LintResults{
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
		expected := "\x1b[4mtest1\x1b[0m:\n   \x1b[31m✖\x1b[0m some error\n   \x1b[31m✖\x1b[0m some error2\n\n\x1b[4mtest2\x1b[0m:\n   \x1b[31m✖\x1b[0m some error\n\n\x1b[31m✖ 3 linting errors found!\x1b[0m\n"
		assert.Equal(t, expected, msg)
	})
}
