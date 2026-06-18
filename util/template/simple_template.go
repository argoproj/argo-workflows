package template

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/errors"
)

var matchAllRegex = regexp.MustCompile(".*")

func simpleReplaceStrict(w io.Writer, tag string, replaceMap map[string]interface{}, strictRegex *regexp.Regexp, allowUnresolvedArtifacts bool) (int, error) {
	replacement, ok := replaceMap[strings.TrimSpace(tag)]
	if !ok {
		// Attempt to resolve nested tags, if possible
		if index := strings.LastIndex(tag, "{{"); index > 0 {
			nestedTagPrefix := tag[:index]
			nestedTag := tag[index+2:]
			if replNested, replOk := replaceMap[nestedTag]; replOk {
				if replNested == nil {
					// A present-but-nil nested value is an absent optional (a skipped node's output
					// with no default): the composite outer tag cannot meaningfully resolve from an
					// absent value. Fail terminally — the message must NOT match IsMissingVariableErr,
					// which would requeue forever.
					return 0, errors.Errorf(errors.CodeBadRequest, "unable to substitute nested tag {{%s}}: %q is an absent optional", tag, nestedTag)
				}
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
		log.WithError(errors.InternalError("unresolved")).Debug("unresolved is allowed")
		return fmt.Fprintf(w, "{{%s}}", tag)
	}

	if replacement == nil {
		// A present-but-nil value is a genuinely absent optional (a skipped/omitted node's output
		// with no default) that nothing handled: no producer valueFrom.default, no consumer input
		// default (which would have dropped the argument before substitution), no expression `??`
		// fallback. Fail terminally — the message must NOT match IsMissingVariableErr, which would
		// requeue forever.
		return 0, errors.Errorf(errors.CodeBadRequest, "unable to substitute {{%s}}: output is an absent optional (skipped/omitted node with no default)", tag)
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
