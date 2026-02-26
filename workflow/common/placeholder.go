package common

import "github.com/argoproj/argo-workflows/v4/util/template"

// placeholderGenerator generates dynamically-indexed placeholder strings.
type placeholderGenerator struct {
	index int
}

// NewPlaceholderGenerator returns a placeholderGenerator.
func NewPlaceholderGenerator() *placeholderGenerator {
	return &placeholderGenerator{}
}

// NextPlaceholder returns an arbitrary string to perform mock substitution of variables.
func (p *placeholderGenerator) NextPlaceholder() string {
	s := template.NewPlaceholder(p.index)
	p.index = p.index + 1
	return s
}

func (p *placeholderGenerator) IsPlaceholder(s string) bool {
	return template.IsPlaceholder(s)
}
