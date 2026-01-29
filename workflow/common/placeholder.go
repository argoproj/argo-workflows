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
	s := fmt.Sprintf("placeholder-%d", p.index)
	p.index++
	return s
}

func (p *placeholderGenerator) IsPlaceholder(s string) bool {
	return strings.HasPrefix(s, "placeholder-")
}
