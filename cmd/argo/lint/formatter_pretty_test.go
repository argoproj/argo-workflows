package lint

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/TwiN/go-color"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
)

func TestPrettySummarize(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		msg := formatterPretty{}.Summarize(&LintResults{
			Success: true,
		})
		expected := fmt.Sprintf("%s no linting errors found!\n", color.Ize(color.Green, "✔"))
		assert.Equal(t, expected, msg)
	})
	t.Run("Nothing linted", func(t *testing.T) {
		msg := formatterPretty{}.Summarize(&LintResults{
			anythingLinted: false,
			Success:        false,
		})
		expected := fmt.Sprintf("%s\n", color.Ize(color.Red, "✖ found nothing to lint in the specified paths, failing..."))
		assert.Equal(t, expected, msg)
	})
}

func TestPrettyFormat(t *testing.T) {
	t.Run("Multiple", func(t *testing.T) {
		msg := formatterPretty{}.Format(&LintResult{
			File: "test1",
			Errs: []error{
				fmt.Errorf("some error"),
				fmt.Errorf("some error2"),
			},
			Linted: true,
		})
		expected := "\x1b[4mtest1\x1b[0m:\n   \x1b[31m✖\x1b[0m some error\n   \x1b[31m✖\x1b[0m some error2\n\n"
		if runtime.GOOS == "windows" {
			expected = "test1:\n   ✖ some error\n   ✖ some error2\n\n"
		}
		assert.Equal(t, expected, msg)
	})

	t.Run("One", func(t *testing.T) {
		msg := formatterPretty{}.Format(&LintResult{
			File: "test2",
			Errs: []error{
				fmt.Errorf("some error"),
			},
			Linted: true,
		})
		expected := "\x1b[4mtest2\x1b[0m:\n   \x1b[31m✖\x1b[0m some error\n\n"
		if runtime.GOOS == "windows" {
			expected = "test2:\n   ✖ some error\n\n"
		}
		assert.Equal(t, expected, msg)
	})

	t.Run("NotLinted", func(t *testing.T) {
		msg := formatterPretty{}.Format(&LintResult{
			File:   "test3",
			Linted: false,
		})
		expected := ""
		assert.Equal(t, expected, msg)
	})
}

func TestPrettySummarizeWithColorDisabled(t *testing.T) {
	common.NoColor = true
	defer func() {
		common.NoColor = false
	}()

	t.Run("Success", func(t *testing.T) {
		msg := formatterPretty{}.Summarize(&LintResults{
			Success: true,
		})
		expected := "✔ no linting errors found!\n"
		assert.Equal(t, expected, msg)
	})
	t.Run("Nothing linted", func(t *testing.T) {
		msg := formatterPretty{}.Summarize(&LintResults{
			anythingLinted: false,
			Success:        false,
		})
		expected := "✖ found nothing to lint in the specified paths, failing...\n"
		assert.Equal(t, expected, msg)
	})
}

func TestPrettyFormatWithColorDisabled(t *testing.T) {
	common.NoColor = true
	defer func() {
		common.NoColor = false
	}()

	t.Run("Multiple", func(t *testing.T) {
		msg := formatterPretty{}.Format(&LintResult{
			File: "test1",
			Errs: []error{
				fmt.Errorf("some error"),
				fmt.Errorf("some error2"),
			},
			Linted: true,
		})
		expected := "test1:\n   ✖ some error\n   ✖ some error2\n\n"
		assert.Equal(t, expected, msg)
	})

	t.Run("One", func(t *testing.T) {
		msg := formatterPretty{}.Format(&LintResult{
			File: "test2",
			Errs: []error{
				fmt.Errorf("some error"),
			},
			Linted: true,
		})
		expected := "test2:\n   ✖ some error\n\n"
		assert.Equal(t, expected, msg)
	})

	t.Run("NotLinted", func(t *testing.T) {
		msg := formatterPretty{}.Format(&LintResult{
			File:   "test3",
			Linted: false,
		})
		expected := ""
		assert.Equal(t, expected, msg)
	})
}
