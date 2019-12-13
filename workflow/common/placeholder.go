package common

import "fmt"

// placeholderGenerator is to generate dynamically-generated placeholder strings.
type placeholderGenerator struct {
	index int
}

// NewPlaceholderGenerator returns a placeholderGenerator.
func NewPlaceholderGenerator() *placeholderGenerator {
	return &placeholderGenerator{}
}

// GetPlaceholder returns an arbitrary string to perform mock substitution of variables
func (p *placeholderGenerator) GetPlaceholder() string {
	s := fmt.Sprintf("placeholder-%d", p.index)
	p.index = p.index + 1
	return s
}
