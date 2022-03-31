package authz

import (
	"strings"

	"k8s.io/utils/strings/slices"
)

func ContainsFunc(args ...interface{}) (interface{}, error) {
	return slices.Contains(strings.Split(args[0].(string), ","), args[1].(string)), nil
}
