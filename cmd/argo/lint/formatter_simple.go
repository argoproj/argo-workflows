package lint

import (
	"fmt"
	"strings"
)

type formatterSimple struct{}

func (f formatterSimple) Format(l *LintResults) string {
	if l.Success {
		return "no linting errors found!\n"
	}

	if !l.anythingLinted {
		return "found nothing to lint in the specified paths, failing...\n"
	}

	sb := &strings.Builder{}

	for _, r := range l.Results {
		if !r.Linted {
			continue
		}

		if len(r.Errs) == 0 {
			continue
		}

		for _, e := range r.Errs {
			fmt.Fprintf(sb, "%s: %s\n", r.File, e)
		}
	}

	return sb.String()
}
