package lint

import (
	"fmt"
	"strings"

	"github.com/TwiN/go-color"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/common"
)

const (
	lintIndentation = "   "
)

type formatterPretty struct{}

func (f formatterPretty) Format(l *Result) string {
	setColorize()
	if !l.Linted {
		return ""
	}

	if len(l.Errs) == 0 {
		return ""
	}

	sb := &strings.Builder{}
	fmt.Fprintf(sb, "%s:\n", color.Ize(color.Underline, l.File)) // print source name

	for _, e := range l.Errs {
		fmt.Fprintf(sb, "%s%s %s\n", lintIndentation, color.Ize(color.Red, "✖"), e)
	}
	sb.WriteString("\n")

	return sb.String()
}

func (f formatterPretty) Summarize(l *Results) string {
	setColorize()
	if l.Success {
		return fmt.Sprintf("%s no linting errors found!\n", color.Ize(color.Green, "✔"))
	}

	if !l.anythingLinted {
		return fmt.Sprintf("%s\n", color.Ize(color.Red, "✖ found nothing to lint in the specified paths, failing..."))
	}

	totErr := 0
	for _, r := range l.Results {
		totErr += len(r.Errs)
	}

	return fmt.Sprintln(color.Ize(color.Red, fmt.Sprintf("✖ %d linting errors found!", totErr)))
}

func setColorize() {
	color.Toggle(!common.NoColor)
}
