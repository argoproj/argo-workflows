package dag

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func intstrPtr(s string) *intstr.IntOrString {
	v := intstr.FromString(s)
	return &v
}

// TestExpandSequence_BackwardCounting verifies that withSequence with start > end
// produces items counting backwards (e.g., start=5, end=1 → [5,4,3,2,1]).
// The old code in operator.go explicitly handled this with a reverse loop.
func TestExpandSequence_BackwardCounting(t *testing.T) {
	seq := &wfv1.Sequence{
		Start: intstrPtr("5"),
		End:   intstrPtr("1"),
	}

	items, err := expandSequence(seq)
	require.NoError(t, err)
	require.NotNil(t, items, "withSequence start=5 end=1 should produce items, not nil")
	require.Len(t, items, 5, "should produce 5 items counting from 5 down to 1")

	var values []string
	for _, item := range items {
		values = append(values, string(item.Value))
	}
	assert.Equal(t, []string{`"5"`, `"4"`, `"3"`, `"2"`, `"1"`}, values)
}

// TestExpandSequence_BackwardCountingWithFormat verifies backward counting
// with a custom format string.
func TestExpandSequence_BackwardCountingWithFormat(t *testing.T) {
	seq := &wfv1.Sequence{
		Start:  intstrPtr("3"),
		End:    intstrPtr("0"),
		Format: "item-%02d",
	}

	items, err := expandSequence(seq)
	require.NoError(t, err)
	require.Len(t, items, 4)

	var values []string
	for _, item := range items {
		values = append(values, string(item.Value))
	}
	assert.Equal(t, []string{`"item-03"`, `"item-02"`, `"item-01"`, `"item-00"`}, values)
}

// TestExpandSequence_ProducesStringItems verifies that withSequence without a format
// string produces JSON string items (e.g., "0", "1", "2"), not JSON numbers (0, 1, 2).
// The old code always used ParseItem(`"` + fmt.Sprintf("%d", i) + `"`), which wraps
// the value in quotes, producing string items.
func TestExpandSequence_ProducesStringItems(t *testing.T) {
	seq := &wfv1.Sequence{
		Count: intstrPtr("3"),
	}

	items, err := expandSequence(seq)
	require.NoError(t, err)
	require.Len(t, items, 3)

	for i, item := range items {
		var parsed any
		err := json.Unmarshal(item.Value, &parsed)
		require.NoError(t, err)

		// The value must be a JSON string, not a number.
		// Old behavior: ParseItem(`"0"`) → JSON string "0"
		// Bug behavior: json.Marshal(0) → JSON number 0
		strVal, ok := parsed.(string)
		assert.True(t, ok, "item %d should be a JSON string, got %T: %s", i, parsed, string(item.Value))
		assert.Equal(t, string(rune('0'+i)), strVal)
	}
}

// TestExpandSequence_ForwardCounting confirms forward counting still works.
func TestExpandSequence_ForwardCounting(t *testing.T) {
	seq := &wfv1.Sequence{
		Start: intstrPtr("0"),
		End:   intstrPtr("2"),
	}

	items, err := expandSequence(seq)
	require.NoError(t, err)
	require.Len(t, items, 3)

	var values []string
	for _, item := range items {
		values = append(values, string(item.Value))
	}
	assert.Equal(t, []string{`"0"`, `"1"`, `"2"`}, values)
}
