package pool

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// Number is a pool setting that is a whole number, such as
// pool.initialReplicas, written as a number or, in a template, as a string
// that may hold a template variable.
type Number struct {
	// Field names the setting in errors, such as "initialReplicas".
	Field string
	// Min and Max bound its value.
	Min, Max int64
}

var (
	// InitialReplicas is pool.initialReplicas.
	InitialReplicas = Number{Field: "initialReplicas", Min: 1, Max: 1<<31 - 1}
	// RetentionSeconds is pool.retentionSeconds.
	RetentionSeconds = Number{Field: "retentionSeconds", Min: 0, Max: 1<<63 - 1}
)

// HasVariable reports whether v holds a template variable, which is checked
// once substituted.
func HasVariable(v *intstr.IntOrString) bool {
	return v != nil && v.Type == intstr.String && strings.Contains(v.StrVal, "{{")
}

// Value returns v's value, and false if v is unset. It fails if v isn't a
// whole number within the setting's bounds, as a template variable not yet
// substituted isn't.
func (n Number) Value(v *intstr.IntOrString) (int64, bool, error) {
	if v == nil {
		return 0, false, nil
	}
	i := int64(v.IntVal)
	if v.Type == intstr.String {
		var err error
		if i, err = strconv.ParseInt(strings.TrimSpace(v.StrVal), 10, 64); err != nil {
			return 0, false, fmt.Errorf("%s '%s' is not a whole number", n.Field, v.StrVal)
		}
	}
	switch {
	case i < n.Min && n.Min == 0:
		return 0, false, fmt.Errorf("%s must not be negative", n.Field)
	case i < n.Min:
		return 0, false, fmt.Errorf("%s must be at least %d", n.Field, n.Min)
	case i > n.Max:
		return 0, false, fmt.Errorf("%s must be at most %d", n.Field, n.Max)
	}
	return i, true, nil
}
