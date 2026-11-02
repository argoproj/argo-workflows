package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNumberValue(t *testing.T) {
	for _, tt := range []struct {
		name   string
		number Number
		value  *intstr.IntOrString
		want   int64
		set    bool
		err    string
	}{
		{name: "unset", number: InitialReplicas},
		{name: "a number", number: InitialReplicas, value: new(intstr.FromInt32(3)), want: 3, set: true},
		{name: "a string", number: RetentionSeconds, value: new(intstr.FromString(" 60 ")), want: 60, set: true},
		{name: "zero retention", number: RetentionSeconds, value: new(intstr.FromInt32(0)), want: 0, set: true},
		{name: "below the minimum", number: InitialReplicas, value: new(intstr.FromInt32(0)), err: "initialReplicas must be at least 1"},
		{name: "negative", number: RetentionSeconds, value: new(intstr.FromInt32(-1)), err: "retentionSeconds must not be negative"},
		{name: "above the maximum", number: InitialReplicas, value: new(intstr.FromString("2147483648")), err: "initialReplicas must be at most 2147483647"},
		{name: "not a number", number: InitialReplicas, value: new(intstr.FromString("two")), err: "initialReplicas 'two' is not a whole number"},
		{name: "a variable", number: InitialReplicas, value: new(intstr.FromString("{{inputs.parameters.n}}")), err: "is not a whole number"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, set, err := tt.number.Value(tt.value)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.set, set)
		})
	}
}

func TestHasVariable(t *testing.T) {
	assert.False(t, HasVariable(nil))
	assert.False(t, HasVariable(new(intstr.FromInt32(1))))
	assert.False(t, HasVariable(new(intstr.FromString("1"))))
	assert.True(t, HasVariable(new(intstr.FromString("{{workflow.parameters.n}}"))))
}
