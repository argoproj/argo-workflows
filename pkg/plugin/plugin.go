package plugin

import "reflect"

func ID(p interface{}) string {
	return reflect.TypeOf(p).String()
}
