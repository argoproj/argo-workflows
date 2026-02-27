package intstr

import "k8s.io/apimachinery/pkg/util/intstr"

// ParsePtr is a convenience function to parse a string and return a pointer to the result.
func ParsePtr(val string) *intstr.IntOrString {
	x := intstr.Parse(val)
	return &x
}
