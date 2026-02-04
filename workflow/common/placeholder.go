package common

import (
	"fmt"
	"strings"
)

// PlaceholderGenerator is the interface for generating placeholder strings.
type PlaceholderGenerator interface {
	NextPlaceholder() string
	IsPlaceholder(s string) bool
}

// placeholderGenerator is to generate dynamically-generated placeholder strings.
type placeholderGenerator struct {
	index int
}

// NewPlaceholderGenerator returns a PlaceholderGenerator.
func NewPlaceholderGenerator() PlaceholderGenerator {
	return &placeholderGenerator{}
}

// NextPlaceholder returns an arbitrary string to perform mock substitution of variables
func (p *placeholderGenerator) NextPlaceholder() string {
	s := fmt.Sprintf("__argo__internal__placeholder-%d", p.index)
	p.index++
	return s
}

func (p *placeholderGenerator) IsPlaceholder(s string) bool {
	return strings.HasPrefix(s, "__argo__internal__placeholder-")
}
