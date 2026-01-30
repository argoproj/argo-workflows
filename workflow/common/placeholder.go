package common

import (
	"fmt"
	"strings"
)

// placeholderGenerator is to generate dynamically-generated placeholder strings.
type placeholderGenerator struct {
	index int
}

// NewPlaceholderGenerator returns a placeholderGenerator.
func NewPlaceholderGenerator() *placeholderGenerator {
	return &placeholderGenerator{}
}

// NextPlaceholder returns an arbitrary string to perform mock substitution of variables
func (p *placeholderGenerator) NextPlaceholder() string {
	s := fmt.Sprintf("__argo__internal__placeholder-%d", p.index)
	p.index += 1
	return s
}

func (p *placeholderGenerator) IsPlaceholder(s string) bool {
	return strings.HasPrefix(s, "__argo__internal__placeholder-")
}
