package mapper

import (
	"fmt"
	"reflect"
)

// Map translate (maps) any object by recursively applying the mapper func to each field, array element, and map value.
// Among intended use cases is translating a data structure (e.g. from English to Spanish).
func Map(x any, m func(any) (any, error)) (any, error) {
	y, err := _map(reflect.ValueOf(x), m)
	if err != nil {
		return x, fmt.Errorf("failed to map %v: %w", x, err)
	}
	return y.Interface(), nil
}

func _map(x reflect.Value, m func(any) (any, error)) (reflect.Value, error) {
	if x.IsZero() {
		return x, nil
	}
	switch x.Kind() {
	case reflect.Ptr:
		y, err := _map(x.Elem(), m)
		if err != nil {
			return x, fmt.Errorf("failed to map %v: %w", x, err)
		}
		z := reflect.New(y.Type())
		z.Elem().Set(y)
		return z, nil
	case reflect.Struct:
		y := reflect.Indirect(reflect.New(x.Type()))
		for i := 0; i < x.NumField(); i++ {
			if y.Field(i).CanSet() { // we cannot set un-exported values, so leave the as zero-value
				g, err := _map(x.Field(i), m)
				if err != nil {
					return x, fmt.Errorf("failed to map %v: %w", x, err)
				}
				y.Field(i).Set(g)
			}
		}
		return y, nil
	case reflect.Array, reflect.Slice:
		y := reflect.MakeSlice(x.Type(), x.Len(), x.Len())
		for i := 0; i < x.Len(); i++ {
			g, err := _map(x.Index(i), m)
			if err != nil {
				return x, fmt.Errorf("failed to map %v index %d: %w", x, i, err)
			}
			y.Index(i).Set(g)
		}
		return y, nil
	case reflect.Map:
		y := reflect.MakeMap(x.Type())
		for _, key := range x.MapKeys() {
			g, err := _map(x.MapIndex(key), m)
			if err != nil {
				return x, fmt.Errorf("failed to map %v: %w", key, err)
			}
			y.SetMapIndex(key, g)
		}
		return y, nil
	default:
		y, err := m(reflect.Indirect(x).Interface())
		if err != nil {
			return x, fmt.Errorf("failed to map %v: %w", x, err)
		}
		return reflect.ValueOf(y), nil
	}
}
