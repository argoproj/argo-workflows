package template

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

var matchAllRegex = regexp.MustCompile(".*")

func simpleReplaceStrict(ctx context.Context, w io.Writer, tag string, replaceMap map[string]any, strictRegex *regexp.Regexp, allowUnresolvedArtifacts bool) (int, error) {
	replacement, ok := replaceMap[strings.TrimSpace(tag)]
	if !ok {
		// Attempt to resolve nested tags, if possible
		if index := strings.LastIndex(tag, "{{"); index > 0 {
			nestedTagPrefix := tag[:index]
			nestedTag := tag[index+2:]
			if replNested, replOk := replaceMap[nestedTag]; replOk {
				replStr, isStr := replNested.(string)
				if isStr {
					replStr = strconv.Quote(replStr)
					replStr = replStr[1 : len(replStr)-1]
					return w.Write([]byte("{{" + nestedTagPrefix + replStr))
				}
			}
		}

		// Strict check: if the tag starts with any strict prefix, it MUST be resolved.
		// Exception: Artifacts (containing ".outputs.artifacts.") are handled later by artifact resolution logic
		// and support "optional: true", so we allow them to remain unresolved here.
		trimmedTag := strings.TrimSpace(tag)
		isStrict := strictRegex != nil && strictRegex.MatchString(trimmedTag)

		if isStrict && (!allowUnresolvedArtifacts || !strings.Contains(trimmedTag, ".outputs.artifacts.")) {
			return 0, errors.Errorf(errors.CodeBadRequest, "failed to resolve {{%s}}", tag)
		}

		// allow unresolved (implied true for non-strict tags or artifacts)
		// just write the same string back
		logger := logging.RequireLoggerFromContext(ctx)
		logger.WithError(errors.InternalError("unresolved")).Debug(ctx, "unresolved is allowed")
		return fmt.Fprintf(w, "{{%s}}", tag)
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
