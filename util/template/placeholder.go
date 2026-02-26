package template

import (
	"fmt"
	"strings"
)

const placeholderPrefix = "__argo__internal__placeholder-"

// NewPlaceholder generates an internal Argo placeholder string for the given index.
func NewPlaceholder(index int) string {
	return fmt.Sprintf("%s%d", placeholderPrefix, index)
}

// IsPlaceholder reports whether s is an internal Argo placeholder value.
func IsPlaceholder(s string) bool {
	return strings.HasPrefix(s, placeholderPrefix)
}
