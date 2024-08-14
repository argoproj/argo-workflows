package lint

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleSummarize(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		msg := formatterSimple{}.Summarize(&LintResults{
			Success: true,
		})
		expected := "no linting errors found!\n"
		require.Equal(t, expected, msg)
	})
	t.Run("Nothing linted", func(t *testing.T) {
		msg := formatterSimple{}.Summarize(&LintResults{
			anythingLinted: false,
			Success:        false,
		})
		expected := "found nothing to lint in the specified paths, failing...\n"
		require.Equal(t, expected, msg)
	})
}

func TestSimpleFormat(t *testing.T) {
	t.Run("Multiple", func(t *testing.T) {
		msg := formatterSimple{}.Format(&LintResult{
			File: "test1",
			Errs: []error{
				fmt.Errorf("some error"),
				fmt.Errorf("some error2"),
			},
			Linted: true,
		})
		expected := `test1: some error
test1: some error2
`
		require.Equal(t, expected, msg)
	})

	t.Run("One", func(t *testing.T) {
		msg := formatterSimple{}.Format(&LintResult{
			File: "test2",
			Errs: []error{
				fmt.Errorf("some error"),
			},
			Linted: true,
		})
		expected := "test2: some error\n"
		require.Equal(t, expected, msg)
	})

	t.Run("NotLinted", func(t *testing.T) {
		msg := formatterSimple{}.Format(&LintResult{
			File:   "test3",
			Linted: false,
		})
		expected := ""
		require.Equal(t, expected, msg)
	})
}
