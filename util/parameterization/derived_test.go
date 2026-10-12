package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDerivedSourceOrderConflictsAndCycles(t *testing.T) {
	model := sampleModel()
	model.Derived = []Derivation{{Path: "/second", From: "/first"}, {Path: "/first", From: "/count"}}
	value, sources, errs := Resolve(model, map[string]any{"count": 0})
	require.Empty(t, errs)
	require.InDelta(t, float64(0), value.(map[string]any)["second"], 0)
	require.Equal(t, "derived:/first", sources["/second"])
	_, _, errs = Resolve(model, map[string]any{"count": 0, "first": 9})
	require.Equal(t, "InvalidDerivation", errs[0].Code)
	model.Derived = []Derivation{{Path: "/first", From: "/second"}, {Path: "/second", From: "/first"}}
	_, _, errs = Resolve(model, map[string]any{})
	require.Contains(t, errs[0].Message, "cycle")
}
