package mapper

import (
	"reflect"
)

// Map translate (maps) any object by recursively applying the mapper func to each field, array element, and map value.
// Among intended use cases is translating a data structure (e.g. from English to Spanish).
func Map(x any, m func(any) (any, error)) (any, error) {
	value, err := _map(reflect.ValueOf(x), m)
	return value.Interface(), err
}

func _map(x reflect.Value, m func(any) (any, error)) (reflect.Value, error) {
	if x.IsZero() {
		return x, nil
	}
	switch x.Kind() {
	case reflect.Ptr:
		y, err := _map(x.Elem(), m)
		return y.Addr(), err
	case reflect.Struct:
		y := reflect.Indirect(reflect.New(x.Type()))
		for i := 0; i < x.NumField(); i++ {
			g, err := _map(x.Field(i), m)
			if err != nil {
				return y, err
			}
			y.Field(i).Set(g)
		}
		return y, nil
	case reflect.Array, reflect.Slice:
		y := reflect.Indirect(reflect.MakeSlice(x.Type(), x.Len(), x.Len()))
		for i := 0; i < x.Len(); i++ {
			g, err := _map(x.Index(i), m)
			if err != nil {
				return y, err
			}
			y.Index(i).Set(g)
		}
		return y, nil
	case reflect.Map:
		y := reflect.Indirect(reflect.MakeMap(x.Type()))
		for _, key := range x.MapKeys() {
			g, err := _map(x.MapIndex(key), m)
			if err != nil {
				return y, err
			}
			y.SetMapIndex(key, g)
		}
		return y, nil
	default:
		y, err := m(x.Interface())
		return reflect.ValueOf(y), err
	}
}
