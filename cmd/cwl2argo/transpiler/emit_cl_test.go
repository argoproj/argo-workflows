package transpiler

import (
	"testing"
)

func TestSortInputBindingsByPosition(t *testing.T) {
	position4 := 4
	position1 := 1
	position5 := 5
	position3 := 3
	position2 := 2
	bindings := []bindingTuple{
		{CommandlineBinding{}, CWLStringKind, "one", "1"},
		{CommandlineBinding{}, CWLStringKind, "two", "2"},
		{CommandlineBinding{Position: &position1}, CWLStringKind, "three", "3"},
		{CommandlineBinding{}, CWLStringKind, "four", "4"},
		{CommandlineBinding{Position: &position2}, CWLStringKind, "five", "5"},
		{CommandlineBinding{}, CWLStringKind, "six", "6"},
		{CommandlineBinding{Position: &position3}, CWLStringKind, "seven", "7"},
		{CommandlineBinding{Position: &position4}, CWLStringKind, "eight", "8"},
		{CommandlineBinding{Position: &position5}, CWLStringKind, "nine", "9"},
		{CommandlineBinding{}, CWLStringKind, "ten", "10"}}
	sortInputBindingPairsByPosition(bindings)

	var last *int
	last = nil

	for _, pair := range bindings {
		if last == nil {
			last = pair.commandlineBinding.Position
			continue
		}
		if *last+1 != *pair.commandlineBinding.Position {
			t.Errorf("Sorting not correct, expected monotonic sequence [%d,%d] is not monotonic", *last, *pair.commandlineBinding.Position)
		}
		last = pair.commandlineBinding.Position
	}
}
