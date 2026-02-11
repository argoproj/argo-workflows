//go:build tracing

package fixtures

import (
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

// SpanTree provides a hierarchical view of collected spans for verification
type SpanTree struct {
	spans    []CollectedSpan
	bySpanID map[string]*CollectedSpan
	children map[string][]*CollectedSpan
	roots    []*CollectedSpan
	byName   map[string][]*CollectedSpan
}

// BuildSpanTree creates a SpanTree from a list of collected spans
func BuildSpanTree(spans []CollectedSpan) *SpanTree {
	tree := &SpanTree{
		spans:    spans,
		bySpanID: make(map[string]*CollectedSpan),
		children: make(map[string][]*CollectedSpan),
		roots:    make([]*CollectedSpan, 0),
		byName:   make(map[string][]*CollectedSpan),
	}

	// Index spans by ID
	for i := range spans {
		span := &spans[i]
		tree.bySpanID[span.SpanID] = span
		tree.byName[span.Name] = append(tree.byName[span.Name], span)
	}

	// Build parent-child relationships
	for i := range spans {
		span := &spans[i]
		if span.ParentSpanID == "" {
			tree.roots = append(tree.roots, span)
		} else {
			tree.children[span.ParentSpanID] = append(tree.children[span.ParentSpanID], span)
		}
	}

	return tree
}

// FindAllByName returns all spans with the given name
func (t *SpanTree) FindAllByName(name string) []*CollectedSpan {
	return t.byName[name]
}

// GetRoots returns all root spans (spans with no parent)
func (t *SpanTree) GetRoots() []*CollectedSpan {
	return t.roots
}

// GetChildren returns all direct children of the given span
func (t *SpanTree) GetChildren(span *CollectedSpan) []*CollectedSpan {
	if span == nil {
		return nil
	}
	return t.children[span.SpanID]
}

// PrintTree returns a string representation of the span tree for debugging
func (t *SpanTree) PrintTree() string {
	var sb strings.Builder
	for _, root := range t.roots {
		t.printSpan(&sb, root, 0)
	}
	return sb.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func (t *SpanTree) printSpan(sb *strings.Builder, span *CollectedSpan, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(sb, "%s- %s (span_id=%s, trace_id=%s)\n",
		indent, span.Name, truncate(span.SpanID, 8), truncate(span.TraceID, 8))
	for _, child := range t.GetChildren(span) {
		t.printSpan(sb, child, depth+1)
	}
}

// GetSpanNames returns a list of all unique span names in the tree
func (t *SpanTree) GetSpanNames() []string {
	names := make([]string, 0, len(t.byName))
	for name := range t.byName {
		names = append(names, name)
	}
	return names
}

// HasSpan returns true if a span with the given name exists
func (t *SpanTree) HasSpan(name string) bool {
	_, ok := t.byName[name]
	return ok
}

// HierarchyError represents a single hierarchy violation
type HierarchyError struct {
	SpanName       string
	ChildName      string
	ExpectedParent string
	ActualParent   string
	Message        string
}

func (e HierarchyError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("span %q has unexpected parent: expected %q, got %q", e.ChildName, e.ExpectedParent, e.ActualParent)
}

// HierarchyErrors is a collection of hierarchy violations
type HierarchyErrors []HierarchyError

func (e HierarchyErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d hierarchy violation(s):\n", len(e)))
	for _, err := range e {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}

// VerifyHierarchy verifies that all collected spans conform to the expected
// parent-child relationships defined in the telemetry package.
// It checks that for each parent span, its children have the correct parent.
// Returns nil if all spans conform, otherwise returns error with details.
func (t *SpanTree) VerifyHierarchy(rootSpans ...*telemetry.Span) error {
	var errors HierarchyErrors

	for _, rootSpan := range rootSpans {
		errors = append(errors, t.verifySpanHierarchy(rootSpan)...)
	}

	if len(errors) == 0 {
		return nil
	}
	return errors
}

// isKnownSpan checks if a span name is defined in our telemetry package
func isKnownSpan(name string) bool {
	// Check root spans
	for _, span := range telemetry.Root {
		if span.RuntimeName() == name || span.FindDescendant(name) != nil {
			return true
		}
	}
	// Check anyParent spans
	for _, span := range telemetry.AnyParent {
		if span.RuntimeName() == name {
			return true
		}
	}
	return false
}

// verifySpanHierarchy recursively verifies that collected spans match expected hierarchy
func (t *SpanTree) verifySpanHierarchy(expectedSpan *telemetry.Span) HierarchyErrors {
	var errors HierarchyErrors

	spanName := expectedSpan.RuntimeName()
	collectedSpans := t.FindAllByName(spanName)

	// For each collected span with this name, verify its children
	for _, collectedSpan := range collectedSpans {
		// Get the actual children of this collected span
		actualChildren := t.GetChildren(collectedSpan)

		// For each actual child, verify it's an expected child type
		for _, actualChild := range actualChildren {
			// Skip unknown/external spans (e.g., "HTTP POST" from k8s client)
			if !isKnownSpan(actualChild.Name) {
				continue
			}

			if !expectedSpan.HasChild(actualChild.Name) {
				// Check if it's an "anyParent" span (like waitClientRateLimiter)
				// These can appear under any parent, so we don't report them as errors
				isAnyParent := false
				for _, anyParentSpan := range telemetry.AnyParent {
					if anyParentSpan.RuntimeName() == actualChild.Name {
						isAnyParent = true
						break
					}
				}
				if !isAnyParent {
					errors = append(errors, HierarchyError{
						SpanName:       spanName,
						ChildName:      actualChild.Name,
						ExpectedParent: "", // This child shouldn't exist under this parent
						ActualParent:   spanName,
						Message:        fmt.Sprintf("span %q has unexpected child %q (expected children: %v)", spanName, actualChild.Name, expectedSpan.ExpectedChildren()),
					})
				}
			}
		}
	}

	// Recursively verify children
	for _, expectedChild := range expectedSpan.Children() {
		errors = append(errors, t.verifySpanHierarchy(expectedChild)...)
	}

	return errors
}

// VerifyWorkflowHierarchy is a convenience method that verifies the hierarchy
// starting from the workflow root span.
func (t *SpanTree) VerifyWorkflowHierarchy() error {
	return t.VerifyHierarchy(&telemetry.SpanWorkflow)
}
