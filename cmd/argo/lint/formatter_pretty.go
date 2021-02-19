package lint

import (
	"fmt"
	"strings"
)

const (
	lintIndentation = "   "
)

type formatterPretty struct{}

const (
	escape    = "\x1b"
	noFormat  = 0
	underline = "4"
	fgRed     = "31"
	fgGreen   = "32"
)

func withAttribute(s, color string) string {
	return fmt.Sprintf("%s[%sm%s%s[%dm", escape, color, s, escape, noFormat)
}

func (f formatterPretty) Format(l *LintResults) string {
	if l.Success {
		return fmt.Sprintf("%s no linting errors found!\n", withAttribute("✔", fgGreen))
	}

	if !l.anythingLinted {
		return fmt.Sprintf("%s\n", withAttribute("✖ found nothing to lint in the specified paths, failing...", fgRed))
	}

	sb := &strings.Builder{}
	totErr := 0

	for _, r := range l.Results {
		if !r.Linted {
			continue
		}

		nErr := len(r.Errs)
		if nErr == 0 {
			continue
		}

		fmt.Fprintf(sb, "%s:\n", withAttribute(r.File, underline)) // print source name

		totErr += nErr
		for _, e := range r.Errs {
			fmt.Fprintf(sb, "%s%s %s\n", lintIndentation, withAttribute("✖", fgRed), e)
		}

		fmt.Fprintln(sb)
	}

	fmt.Fprintf(sb, "%s\n", withAttribute(fmt.Sprintf("✖ %d linting errors found!", totErr), fgRed))

	return sb.String()
}
