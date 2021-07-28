package typeassert

import (
	"fmt"
	"reflect"
)

// Type asserter for jwt Claims.
// Some of these claims are optional and
// we return zero values for an empty claim

// Check if interface{} is boolean and return bool if it is
func Bool(i interface{}) (bool, error) {
	if i == nil {
		return false, nil
	}
	v, ok := i.(bool)
	if !ok {
		return false, fmt.Errorf("Interface is not a bool")
	}
	return v, nil
}

// Check if interface{} is float64 and return float64 if it is
func Float64(i interface{}) (float64, error) {
	if i == nil {
		return 0.0, nil
	}
	v, ok := i.(float64)
	if !ok {
		return 0.0, fmt.Errorf("Interface is not a float64")
	}
	return v, nil
}

// Check if interface{} is string and return string if it is
func String(i interface{}) (string, error) {
	if i == nil {
		return "", nil
	}
	v, ok := i.(string)
	if !ok {
		return "", fmt.Errorf("Interface is not a string")
	}
	return v, nil
}

// Check if interface{} is slice of strings and return slice of strings if it is
func StringSlice(i interface{}) ([]string, error) {
	if i == nil {
		// avoid NPE
		return []string{}, nil
	}

	sliceInterface, ok := i.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Expected a slice of interfaces, got %v", reflect.TypeOf(i))
	}
	newSlice := []string{}
	for _, a := range sliceInterface {
		val, err := String(a)
		if err != nil {
			return nil, err
		}
		newSlice = append(newSlice, val)
	}

	return newSlice, nil
}
