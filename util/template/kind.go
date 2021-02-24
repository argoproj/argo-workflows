package template

import (
	"strings"

	jsonutil "github.com/argoproj/argo-workflows/v3/util/json"
)

type kind = string // defines the prefix symbol that determines the syntax of the tag

var kinds []kind

func parseTag(tag string) (kind, string) {
	for _, k := range kinds {
		if strings.HasPrefix(tag, k) {
			return k, jsonutil.Fix(strings.TrimPrefix(tag, k))
		}
	}
	return kindSimple, tag
}
