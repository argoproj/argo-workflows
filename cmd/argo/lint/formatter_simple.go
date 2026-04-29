package lint

import (
	"fmt"
	"strings"
)

type formatterSimple struct{}

func (f formatterSimple) Format(l *Result) string {
	if !l.Linted {
		return ""
	}

	if len(l.Errs) == 0 {
		return ""
	}

	sb := &strings.Builder{}
	for _, e := range l.Errs {
		fmt.Fprintf(sb, "%s: %s\n", l.File, e)
	}

	return sb.String()
}

func (f formatterSimple) Summarize(l *Results) string {
	if l.Success {
		return "no linting errors found!\n"
	}

	if !l.anythingLinted {
		return "found nothing to lint in the specified paths, failing...\n"
	}

	return ""
}
