package main

import (
	"testing"
)

func TestValidateSpanParentage(t *testing.T) {
	tests := []struct {
		name        string
		spans       spansList
		spanToCheck string
		wantErrors  int
	}{
		{
			name: "span is a trace (root span)",
			spans: spansList{
				{common: common{Name: "RootSpan"}, Root: true},
			},
			spanToCheck: "RootSpan",
			wantErrors:  0,
		},
		{
			name: "span with valid parent chain to trace",
			spans: spansList{
				{common: common{Name: "ChildSpan"}, Parents: []string{"RootSpan"}},
				{common: common{Name: "RootSpan"}, Root: true},
			},
			spanToCheck: "ChildSpan",
			wantErrors:  0,
		},
		{
			name: "span with multi-level parent chain to trace",
			spans: spansList{
				{common: common{Name: "GrandchildSpan"}, Parents: []string{"ChildSpan"}},
				{common: common{Name: "ChildSpan"}, Parents: []string{"RootSpan"}},
				{common: common{Name: "RootSpan"}, Root: true},
			},
			spanToCheck: "GrandchildSpan",
			wantErrors:  0,
		},
		{
			name: "span with missing parent",
			spans: spansList{
				{common: common{Name: "OrphanSpan"}, Parents: []string{"NonExistent"}},
			},
			spanToCheck: "OrphanSpan",
			wantErrors:  1,
		},
		{
			name: "span with no parent declared and not a trace",
			spans: spansList{
				{common: common{Name: "OrphanSpan"}},
			},
			spanToCheck: "OrphanSpan",
			wantErrors:  1,
		},
		{
			name: "cycle detection - self reference",
			spans: spansList{
				{common: common{Name: "SelfRef"}, Parents: []string{"SelfRef"}},
			},
			spanToCheck: "SelfRef",
			wantErrors:  1, // at minimum, cycle error
		},
		{
			name: "cycle detection - two span cycle",
			spans: spansList{
				{common: common{Name: "SpanA"}, Parents: []string{"SpanB"}},
				{common: common{Name: "SpanB"}, Parents: []string{"SpanA"}},
			},
			spanToCheck: "SpanA",
			wantErrors:  1, // at minimum, cycle error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v validator
			var spanToCheck span
			for _, s := range tt.spans {
				if s.Name == tt.spanToCheck {
					spanToCheck = s
					break
				}
			}
			v.validateSpanParentage(spanToCheck, []string{}, &tt.spans)
			if len(v.errors) < tt.wantErrors {
				t.Errorf("validateSpanParentage() got %d errors, want at least %d", len(v.errors), tt.wantErrors)
				for _, err := range v.errors {
					t.Logf("  error: %v", err)
				}
			}
		})
	}
}
