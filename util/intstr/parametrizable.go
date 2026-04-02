package intstr

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// These are utility functions when using IntOrString to hold either an Int or an Argo Variable
func Int(is *intstr.IntOrString) (*int, error) {
	if is == nil {
		return nil, nil
	}
	if is.Type == intstr.String {
		i, err := strconv.Atoi(is.StrVal)
		if err != nil {
			return nil, fmt.Errorf("value '%s' cannot be resolved to an int", is.StrVal)
		}
		return &i, nil
	}
	i := int(is.IntVal)
	return &i, nil
}

func Int32(is *intstr.IntOrString) (*int32, error) {
	v, err := Int(is)
	if v == nil || err != nil {
		return nil, err
	}
	i := int32(*v)
	return &i, err
}

func Int64(is *intstr.IntOrString) (*int64, error) {
	v, err := Int(is)
	if v == nil || err != nil {
		return nil, err
	}
	i := int64(*v)
	return &i, err
}

func IsValidIntOrArgoVariable(is *intstr.IntOrString) bool {
	if is == nil || is.Type == intstr.Int {
		return true
	}
	if _, err := strconv.Atoi(is.StrVal); err == nil {
		return true
	}
	return strings.HasPrefix(is.StrVal, "{{") && strings.HasSuffix(is.StrVal, "}}")
}
