package common

import "github.com/argoproj/argo-workflows/v4/util/template"

// PlaceholderGenerator is the interface for generating placeholder strings.
type PlaceholderGenerator interface {
	NextPlaceholder() string
	IsPlaceholder(s string) bool
}

// placeholderGenerator generates dynamically-indexed placeholder strings.
type placeholderGenerator struct {
	index int
}

// NewPlaceholderGenerator returns a PlaceholderGenerator.
func NewPlaceholderGenerator() PlaceholderGenerator {
	return &placeholderGenerator{}
}

// NextPlaceholder returns an arbitrary string to perform mock substitution of variables.
func (p *placeholderGenerator) NextPlaceholder() string {
	s := template.NewPlaceholder(p.index)
	p.index++
	return s
}

func (p *placeholderGenerator) IsPlaceholder(s string) bool {
	return template.IsPlaceholder(s)
}
