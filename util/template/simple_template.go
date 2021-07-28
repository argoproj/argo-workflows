package template

import (
	"fmt"
	"io"
	"strings"

	"github.com/argoproj/argo-workflows/v3/errors"
)

func simpleReplace(w io.Writer, tag string, replaceMap map[string]string, allowUnresolved bool) (int, error) {
	replacement, ok := replaceMap[strings.TrimSpace(tag)]
	if !ok {
		// Attempt to resolve nested tags, if possible
		if index := strings.LastIndex(tag, "{{"); index > 0 {
			nestedTagPrefix := tag[:index]
			nestedTag := tag[index+2:]
			if replacement, ok := replaceMap[nestedTag]; ok {
				return w.Write([]byte("{{" + nestedTagPrefix + replacement))
			}
		}
		if allowUnresolved {
			// just write the same string back
			return w.Write([]byte(fmt.Sprintf("{{%s}}", tag)))
		}
		return 0, errors.Errorf(errors.CodeBadRequest, "failed to resolve {{%s}}", tag)
	}
	return w.Write([]byte(replacement))
}
