package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A span shared between traces lists every trace's spans as parents; each
// trace's diagram draws only the edges from its own spans.
func TestGenerateMermaidSpanTree_OnlyParentsInTrace(t *testing.T) {
	spans := spansList{
		{common: common{Name: "RootA"}, Root: true},
		{common: common{Name: "RootB"}, Root: true},
		{common: common{Name: "Shared"}, Parents: []string{"RootA", "RootB"}},
	}
	lines := generateMermaidSpanTree("RootA", &spans)
	assert.Contains(t, lines, "    RootA --> Shared")
	for _, l := range lines {
		assert.NotContains(t, l, "RootB")
	}
}
