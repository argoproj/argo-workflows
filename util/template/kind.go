package template

import (
	"strings"

	jsonutil "github.com/argoproj/argo-workflows/v3/util/json"
)

type kind = string // defines the prefix symbol that determines the syntax of the tag

const (
	kindSimple     kind = "" // default is simple, i.e. no prefix
	kindExpression kind = "="
)

var kinds []kind

func registerKind(k kind) {
	kinds = append(kinds, k)
}

func parseTag(tag string) (kind, string) {
	for _, k := range kinds {
		if strings.HasPrefix(tag, k) {
			return k, jsonutil.Fix(strings.TrimPrefix(tag, k))
		}
	}
	return kindSimple, tag
}
