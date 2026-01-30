package maps

import (
	"reflect"
	"strconv"
)

func castToMapStringAny(in interface{}) (map[string]interface{}, bool) {
	if m, ok := in.(map[string]interface{}); ok {
		return m, true
	}
	v := reflect.ValueOf(in)
	if v.Kind() != reflect.Map {
		return nil, false
	}
	if v.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	out := make(map[string]interface{}, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		val := iter.Value()
		if !val.IsValid() {
			out[iter.Key().String()] = nil
		} else {
			out[iter.Key().String()] = val.Interface()
		}
	}
	return out, true
}

func castToSliceAny(in interface{}) ([]interface{}, bool) {
	if s, ok := in.([]interface{}); ok {
		return s, true
	}
	v := reflect.ValueOf(in)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, false
	}
	out := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		val := v.Index(i)
		if !val.IsValid() {
			out[i] = nil
		} else {
			out[i] = val.Interface()
		}
	}
	return out, true
}

func VisitArray(as []any, visitor func(key string, value any) bool) bool {
	for i, a := range as {
		var cont bool
		if a == nil {
			cont = visitor(strconv.Itoa(i), a)
			if !cont {
				return false
			}
			continue
		}
		switch reflect.TypeOf(a).Kind() {
		case reflect.Map:
			if m, ok := castToMapStringAny(a); ok {
				cont = VisitMap(m, visitor)
			} else {
				cont = visitor(strconv.Itoa(i), a)
			}
		case reflect.Slice, reflect.Array:
			if s, ok := castToSliceAny(a); ok {
				cont = VisitArray(s, visitor)
			} else {
				cont = visitor(strconv.Itoa(i), a)
			}
		default:
			cont = visitor(strconv.Itoa(i), a)
		}
		if !cont {
			return false
		}
	}
	return true
}

func VisitMap(m map[string]any, visitor func(key string, value any) bool) bool {
	for key, value := range m {
		var cont bool
		if value == nil {
			cont = visitor(key, value)
			if !cont {
				return false
			}
			continue
		}
		switch reflect.TypeOf(value).Kind() {
		case reflect.Map:
			if nestedMap, ok := castToMapStringAny(value); ok {
				cont = VisitMap(nestedMap, visitor)
			} else {
				cont = visitor(key, value)
			}
		case reflect.Slice, reflect.Array:
			if nestedSlice, ok := castToSliceAny(value); ok {
				cont = VisitArray(nestedSlice, visitor)
			} else {
				cont = visitor(key, value)
			}
		default:
			cont = visitor(key, value)
		}
		if !cont {
			return cont
		}
	}
	return true
}
