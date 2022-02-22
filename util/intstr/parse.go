package intstr

import "k8s.io/apimachinery/pkg/util/intstr"

// convenience func to get a pointer
func ParsePtr(val string) *intstr.IntOrString {
	x := intstr.Parse(val)
	return &x
}
