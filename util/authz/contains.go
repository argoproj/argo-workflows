package authz

import "strings"

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func containsFunc(args ...interface{}) (interface{}, error) {
	return contains(strings.Split(args[0].(string), ","), args[1].(string)), nil
}
