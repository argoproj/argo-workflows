package intstr

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
)

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

func Int64(is *intstr.IntOrString) (*int64, error) {
	v, err := Int(is)
	if v == nil {
		return nil, nil
	}
	i := int64(*v)
	return &i, err
}

func IsValidIntOrArgoVariable(is *intstr.IntOrString) bool {
	if is == nil {
		return true
	} else if is.Type == intstr.Int {
		return true
	} else if _, err := strconv.Atoi(is.StrVal); err == nil {
		return true
	} else {
		return strings.HasPrefix(is.StrVal, "{{") && strings.HasSuffix(is.StrVal, "}}")
	}
}
